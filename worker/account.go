package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/xyloni/tdm-light/parser"
	"github.com/xyloni/tdm-light/twitch"
)

// accountWorker — горутина одного аккаунта внутри активной группы.
type accountWorker struct {
	m                *Manager
	acc              *parser.Account
	st               *AccountState
	gql              *twitch.GQLClient
	startupDelay     time.Duration
	unavailable      map[string]time.Time
	nextRateRecovery time.Time
}

func (w *accountWorker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	pubsubDone := make(chan struct{})
	defer func() {
		cancel()
		<-pubsubDone
	}()
	defer func() {
		if ctx.Err() != nil && !w.st.authFailed.Load() {
			w.st.setStatus("stopped")
		}
	}()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(w.acc.Line)*7919))

	// Задержка рассчитана менеджером по числу аккаунтов активной группы.
	if !sleepCtx(ctx, w.startupDelay) {
		close(pubsubDone)
		return
	}

	// Опциональный WebSocket-канал событий дропов (по умолчанию выключен).
	if w.m.cfg.UsePubSub {
		go func() {
			defer close(pubsubDone)
			w.gql.PubSubLoop(ctx, func(topic string, msg json.RawMessage) {
				s := string(msg)
				if len(s) > 200 {
					s = s[:200] + "…"
				}
				w.st.addLog("pubsub: " + s)
			})
		}()
	} else {
		close(pubsubDone)
	}

	errCount := 0
	for ctx.Err() == nil {
		drop, ok, err := w.nextDrop(ctx)
		if err != nil {
			errCount++
			if isAuthError(err) {
				w.st.authFailed.Store(true)
				w.st.setStatus("error: auth")
				w.st.addLog("ошибка авторизации, воркер остановлен: " + err.Error())
				return
			}
			w.st.addLog("dashboard: " + err.Error())
			if !sleepCtx(ctx, backoff(errCount, rnd)) {
				return
			}
			continue
		}
		errCount = 0
		if !ok {
			w.st.setStatus("idle")
			w.st.addLog("нет активных дропов по «" + w.m.cfg.TargetGame + "», сессия завершена")
			return
		}
		if err := w.farmDrop(ctx, rnd, drop); err != nil && ctx.Err() == nil {
			w.st.addLog("фарм: " + err.Error())
			if !sleepCtx(ctx, jittered(rnd, 30*time.Second, w.m.cfg.JitterSec)) {
				return
			}
		}
	}
	w.st.setStatus("stopped")
}

// nextDrop — первый невыкупленный и незавершённый дроп по целевой игре.
func (w *accountWorker) nextDrop(ctx context.Context) (twitch.Drop, bool, error) {
	drops, err := w.gql.DropsDashboard(ctx)
	if err != nil {
		return twitch.Drop{}, false, err
	}
	for _, d := range drops {
		if !strings.EqualFold(d.GameName, w.m.cfg.TargetGame) {
			continue
		}
		if d.CampaignStatus != "" && !strings.EqualFold(d.CampaignStatus, "ACTIVE") {
			continue
		}
		if w.m.cfg.DropIgnored(d.Name) {
			continue
		}
		if d.IsClaimed {
			w.markReady(d)
			continue
		}
		if d.Required <= 0 {
			continue
		}
		if d.ProgressKnown && d.Current >= d.Required {
			w.st.setProgress(d)
			w.markReady(d)
			continue
		}
		w.st.mu.Lock()
		w.st.observePendingLocked(d)
		w.st.mu.Unlock()
		return d, true, nil
	}
	return twitch.Drop{}, false, nil
}

// farmDrop набирает время дропа. Получение награды остаётся пользователю.
func (w *accountWorker) farmDrop(ctx context.Context, rnd *rand.Rand, drop twitch.Drop) error {
	return w.farmDropScheduled(ctx, rnd, drop, watchIntervals{
		hls:      20 * time.Second,
		beacon:   time.Duration(maxInt(w.m.cfg.HeartbeatIntervalSec, 60)) * time.Second,
		progress: time.Duration(maxInt(w.m.cfg.CampaignPollIntervalSec, 60)) * time.Second,
	})
}

type watchIntervals struct {
	hls, beacon, progress time.Duration
	timeouts              operationTimeouts
}

// HLS/recovery stays owned by this worker; beacon and Inventory each have one
// sequential background loop. No request goroutines or pending jobs accumulate.
func (w *accountWorker) farmDropScheduled(parent context.Context, rnd *rand.Rand, drop twitch.Drop, intervals watchIntervals) error {
	ctx, cancel := context.WithCancel(parent)
	budgets := intervals.timeouts.defaults()
	w.st.clearWatchRate()
	defer w.st.clearWatchRate()
	w.st.setStatus("farm: " + drop.CampaignName)
	w.st.addLog(fmt.Sprintf("фарм дропа «%s» начат (цель %d мин)", drop.Name, drop.Required))
	w.reportProgress(drop)

	streams := newWatchStream(ctx)
	progress := make(chan inventoryResult, 1)
	var background sync.WaitGroup
	background.Add(2)
	go func() { defer background.Done(); w.beaconLoop(ctx, streams, intervals.beacon, budgets.beacon) }()
	go func() {
		defer background.Done()
		w.inventoryLoop(ctx, drop.ID, intervals.progress, budgets.inventory, progress)
	}()
	defer func() { cancel(); streams.clear(); background.Wait() }()

	variant := ""
	var stream twitch.Stream
	nextHLS := time.Time{}
	nextOpen := time.Time{}
	openFailures := 0
	var rate watchRate
	var rateStarted time.Time
	handleProgress := func(result inventoryResult) bool {
		if result.err != nil {
			w.st.addLog("прогресс: " + result.err.Error())
			return false
		}
		cur := result.drop
		if !result.found {
			w.st.addLog("дроп пропал из дашборда")
			return true
		}
		if cur.IsClaimed {
			w.st.addLog("«" + cur.Name + "» уже получен")
			return true
		}
		w.reportProgress(cur)
		drop = cur
		if cur.ProgressKnown && cur.Current >= cur.Required {
			w.markReady(cur)
			return true
		}
		// A poll that began before the current playback observation window
		// cannot measure that window. Still retain its confirmed progress above.
		if variant == "" || result.started.Before(rateStarted) {
			return false
		}
		slow, gain, elapsed := rate.observe(result.checked, cur.Current, cur.ProgressKnown, 2*intervals.progress+time.Minute)
		w.st.setWatchRate(&rate)
		if slow && w.reserveRateRecovery(result.checked, rnd) {
			w.st.mu.Lock()
			w.st.rateRecoveries++
			w.st.mu.Unlock()
			w.st.clearWatchRate()
			w.st.addLog(fmt.Sprintf("медленное начисление на %s: +%d мин за %.1f мин; восстанавливаю просмотр", stream.Login, gain, elapsed.Minutes()))
			recoveryCtx, stop := context.WithTimeout(ctx, budgets.recovery)
			selected, playlist, err := w.recoverSlowStream(recoveryCtx, rnd, stream)
			stop()
			if ctx.Err() != nil {
				return true
			}
			if err != nil {
				w.st.addLog("восстановление скорости не удалось; продолжаю текущий просмотр")
			} else {
				stream, variant = selected, playlist
				streams.set(stream)
				nextHLS = time.Time{}
			}
			rateStarted = time.Now()
			rate.reset(rateStarted, cur.Current, cur.ProgressKnown)
		}
		return false
	}
	waitForWork := func(deadline time.Time) bool {
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return true
		case result := <-progress:
			return handleProgress(result)
		case <-timer.C:
			return false
		}
	}
	for ctx.Err() == nil {
		// Consume the latest completed poll before doing more network work.
		select {
		case result := <-progress:
			if handleProgress(result) {
				return nil
			}
		default:
		}
		if ctx.Err() != nil {
			return nil
		}
		if variant == "" {
			if time.Now().Before(nextOpen) {
				if waitForWork(nextOpen) {
					return nil
				}
				continue
			}
			recoveryCtx, stop := context.WithTimeout(ctx, budgets.recovery)
			selected, playlist, err := w.openStream(recoveryCtx, rnd)
			stop()
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				if isAuthError(err) {
					return err
				}
				w.st.addLog("поиск канала: " + err.Error())
				openFailures++
				delay := channelRetryDelay(err, openFailures, rnd)
				nextOpen = time.Now().Add(delay)
				w.st.addLog(fmt.Sprintf("следующая попытка выбора канала через %s", delay.Round(time.Second)))
				continue // Inventory continues during the bounded playback retry pause.
			}
			stream, variant = selected, playlist
			rateStarted = time.Now()
			rate.reset(rateStarted, drop.Current, drop.ProgressKnown)
			w.st.clearWatchRate()
			streams.set(stream)
			nextHLS = time.Time{}
			w.st.addLog("просмотр канала: " + stream.Login)
		}
		if !time.Now().Before(nextHLS) {
			started := time.Now()
			checkCtx, stop := context.WithTimeout(ctx, budgets.hls)
			result, err := w.gql.WatchHLS(checkCtx, variant, stream)
			stop()
			if ctx.Err() != nil {
				return nil
			}
			nextHLS = signalDeadline(nextHLS, started, time.Now(), intervals.hls)
			if err != nil {
				w.st.addLog(fmt.Sprintf("канал %s: HLS не восстановлен (%v; %.1f с), переключаю канал", stream.Login, err, time.Since(started).Seconds()))
				duration := w.excludeFailedChannel(stream.Login, err, rnd)
				w.st.addLog(fmt.Sprintf("канал %s: пауза %s", stream.Login, duration))
				streams.clear()
				w.st.clearWatchRate()
				variant = ""
				continue
			}
			if result.VariantURL != "" {
				variant = result.VariantURL
			}
			openFailures = 0
			if result.Warning != "" {
				w.st.addLog("HLS предупреждение: " + result.Warning)
			}
			w.st.addLog(fmt.Sprintf("HLS-проверка выполнена: плейлист и HEAD сегментов (%.1f с)", time.Since(started).Seconds()))
		}
		if waitForWork(nextHLS) {
			return nil
		}
	}
	return nil
}

// Count ready rewards, but announce only a transition from confirmed unfinished
// minutes. Already-ready rewards discovered at startup are silent baselines.
func (w *accountWorker) markReady(drop twitch.Drop) {
	w.st.mu.Lock()
	if w.st.readyDrops == nil {
		w.st.readyDrops = make(map[string]struct{})
	}
	if _, exists := w.st.readyDrops[drop.ID]; exists {
		w.st.mu.Unlock()
		return
	}
	w.st.readyDrops[drop.ID] = struct{}{}
	w.st.DropsReady++
	_, announce := w.st.pendingDrops[drop.ID]
	announce = announce && !drop.IsClaimed
	delete(w.st.pendingDrops, drop.ID)
	w.st.mu.Unlock()
	if announce {
		w.st.addLog(fmt.Sprintf("✅ ВРЕМЯ НАБРАНО: %s (%d/%d мин); получение вручную", drop.Name, drop.Current, drop.Required))
	}
	w.m.recordReady(w.acc, drop, announce)
}

func (w *accountWorker) findDrop(ctx context.Context, dropID string) (twitch.Drop, bool, error) {
	drops, err := w.gql.DropsDashboard(ctx)
	if err != nil {
		return twitch.Drop{}, false, err
	}
	for _, d := range drops {
		if d.ID == dropID {
			return d, true, nil
		}
	}
	return twitch.Drop{}, false, nil
}

// --- helpers ---

func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func sleepUntil(ctx context.Context, deadline time.Time) bool {
	return sleepCtx(ctx, time.Until(deadline))
}

// jittered возвращает base ± случайные [0..jitterSec] секунд.
func jittered(rnd *rand.Rand, base time.Duration, jitterSec int) time.Duration {
	if jitterSec <= 0 {
		return base
	}
	delta := time.Duration(rnd.Intn(2*jitterSec+1)-jitterSec) * time.Second
	d := base + delta
	if d < time.Second {
		d = time.Second
	}
	return d
}

// backoff — экспоненциальная пауза при повторных ошибках (30s..8m + разброс).
func backoff(errCount int, rnd *rand.Rand) time.Duration {
	shift := errCount
	if shift > 4 {
		shift = 4
	}
	d := time.Duration(30*(1<<shift)) * time.Second
	if d > 8*time.Minute {
		d = 8 * time.Minute
	}
	return d + time.Duration(rnd.Intn(30))*time.Second
}

func isAuthError(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "401") ||
		strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "авторизац")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

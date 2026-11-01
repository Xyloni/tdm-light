package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
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
		if d.IsClaimed {
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

type watchIntervals struct{ hls, beacon, progress time.Duration }

// Independent deadlines share the account goroutine; no extra loop per signal.
func (w *accountWorker) farmDropScheduled(ctx context.Context, rnd *rand.Rand, drop twitch.Drop, intervals watchIntervals) error {
	w.st.clearWatchRate()
	defer w.st.clearWatchRate()
	w.st.setStatus("farm: " + drop.CampaignName)
	w.st.addLog("фарм дропа «" + drop.Name + "» начат")
	w.reportProgress(drop)
	variant := ""
	var stream twitch.Stream
	nextHLS, nextBeacon := time.Time{}, time.Time{}
	nextProgress := time.Now().Add(intervals.progress)
	var rate watchRate
	for ctx.Err() == nil {
		if variant == "" {
			selected, playlist, err := w.openStream(ctx, rnd)
			if err != nil {
				return err
			}
			stream, variant = selected, playlist
			rate.reset(time.Now(), drop.Current, drop.ProgressKnown)
			w.st.clearWatchRate()
			nextHLS = time.Time{}
			w.st.addLog("просмотр канала: " + stream.Login)
		}
		if !time.Now().Before(nextHLS) {
			started := time.Now()
			result, err := w.gql.WatchHLS(ctx, variant, stream)
			if ctx.Err() != nil {
				return nil
			}
			nextHLS = signalDeadline(nextHLS, started, time.Now(), intervals.hls)
			if err != nil {
				w.st.addLog("канал " + stream.Login + ": HLS не восстановлен (" + err.Error() + "), переключаю канал")
				w.excludeChannel(stream.Login)
				variant = ""
				continue
			}
			if result.VariantURL != "" {
				variant = result.VariantURL
			}
			if result.Warning != "" {
				w.st.addLog("HLS предупреждение: " + result.Warning)
			}
			w.st.addLog("HLS-проверка выполнена: плейлист и HEAD сегментов")
		}
		if !time.Now().Before(nextBeacon) {
			started := time.Now()
			err := w.gql.SendMinuteWatched(ctx, stream)
			if ctx.Err() != nil {
				return nil
			}
			nextBeacon = signalDeadline(nextBeacon, started, time.Now(), intervals.beacon)
			if err != nil {
				w.st.addLog("minute-watched beacon: " + err.Error())
			} else {
				w.st.addLog("minute-watched beacon отправлен")
			}
		}
		if !time.Now().Before(nextProgress) {
			started := time.Now()
			cur, ok, err := w.findDrop(ctx, drop.ID)
			if ctx.Err() != nil {
				return nil
			}
			nextProgress = signalDeadline(nextProgress, started, time.Now(), intervals.progress)
			if err != nil {
				w.st.addLog("прогресс: " + err.Error())
			} else {
				if !ok {
					w.st.addLog("дроп пропал из дашборда")
					return nil
				}
				if cur.IsClaimed {
					w.st.addLog("«" + cur.Name + "» уже получен")
					return nil
				}
				w.reportProgress(cur)
				drop = cur
				if cur.ProgressKnown && cur.Current >= cur.Required {
					w.markReady(cur)
					return nil
				}
				now := time.Now()
				slow, gain, elapsed := rate.observe(now, cur.Current, cur.ProgressKnown, 2*intervals.progress+time.Minute)
				w.st.setWatchRate(&rate)
				if slow && w.reserveRateRecovery(now, rnd) {
					w.st.mu.Lock()
					w.st.rateRecoveries++
					w.st.mu.Unlock()
					w.st.clearWatchRate()
					// Retained across drops/retries; jitter avoids synchronized recovery.
					w.st.addLog(fmt.Sprintf("медленное начисление на %s: +%d мин за %.1f мин; восстанавливаю просмотр", stream.Login, gain, elapsed.Minutes()))
					selected, playlist, recoveryErr := w.recoverSlowStream(ctx, rnd, stream)
					if ctx.Err() != nil {
						return nil
					}
					if recoveryErr != nil {
						w.st.addLog("восстановление скорости не удалось; продолжаю текущий просмотр")
					} else {
						stream, variant = selected, playlist
						nextHLS = time.Time{}
					}
					// Observe a fresh full window after every attempt, including failure.
					rate.reset(time.Now(), cur.Current, cur.ProgressKnown)
				}
			}
		}
		deadline := nextHLS
		if nextBeacon.Before(deadline) {
			deadline = nextBeacon
		}
		if nextProgress.Before(deadline) {
			deadline = nextProgress
		}
		if !sleepUntil(ctx, deadline) {
			return nil
		}
	}
	return nil
}

// markReady counts and announces each completed drop once per manager lifetime.
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
	w.st.mu.Unlock()
	w.st.addLog("✅ ВРЕМЯ НАБРАНО: " + drop.Name + "; получение вручную")
	w.m.notifyReady(w.acc, drop)
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

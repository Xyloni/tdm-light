package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"dropfarm/parser"
	"dropfarm/twitch"
)

// accountWorker — горутина одного аккаунта внутри активной группы.
type accountWorker struct {
	m            *Manager
	acc          *parser.Account
	st           *AccountState
	gql          *twitch.GQLClient
	startupDelay time.Duration
}

func (w *accountWorker) Run(ctx context.Context) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(w.acc.Line)*7919))

	// Задержка рассчитана менеджером по числу аккаунтов активной группы.
	if !sleepCtx(ctx, w.startupDelay) {
		return
	}

	// Опциональный WebSocket-канал событий дропов (по умолчанию выключен).
	if w.m.cfg.UsePubSub {
		go w.gql.PubSubLoop(ctx, func(topic string, msg json.RawMessage) {
			s := string(msg)
			if len(s) > 200 {
				s = s[:200] + "…"
			}
			w.st.addLog("pubsub: " + s)
		})
	}

	errCount := 0
	for ctx.Err() == nil {
		drop, ok, err := w.nextDrop(ctx)
		if err != nil {
			errCount++
			if isAuthError(err) {
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
		if d.IsClaimed || d.Current >= d.Required {
			continue
		}
		return d, true, nil
	}
	return twitch.Drop{}, false, nil
}

// farmDrop фармит один дроп: выбор стрима → heartbeat-ы → клейм на 100%.
func (w *accountWorker) farmDrop(ctx context.Context, rnd *rand.Rand, drop twitch.Drop) error {
	w.st.setStatus("farm: " + drop.CampaignName)
	w.st.addLog("фарм дропа «" + drop.Name + "» начат")

	variant := ""
	var watchStream twitch.Stream
	lastPoll := time.Now()
	pendingClaim := 0
	hbInterval := time.Duration(maxInt(w.m.cfg.HeartbeatIntervalSec, 20)) * time.Second
	pollInterval := time.Duration(maxInt(w.m.cfg.CampaignPollIntervalSec, 60)) * time.Second
	nextHeartbeat := time.Time{}

	for ctx.Err() == nil {
		// 1. Heartbeat — media playlist + HEAD-запросы к его сегментам и minute beacon.
		if variant != "" {
			heartbeatAt := time.Now()
			result, err := w.gql.WatchHeartbeat(ctx, variant, watchStream)
			if err != nil {
				w.st.addLog("heartbeat оборвался (" + err.Error() + "), переподключение")
				variant = ""
				watchStream = twitch.Stream{}
				nextHeartbeat = time.Time{}
			} else {
				if result.VariantURL != "" {
					variant = result.VariantURL
				}
				if result.Warning != "" {
					w.st.addLog("heartbeat предупреждение: " + result.Warning)
				}
				w.st.addLog("watch heartbeat отправлен: " + result.Method)
				nextHeartbeat = heartbeatDeadline(nextHeartbeat, heartbeatAt, time.Now(), hbInterval)
			}
		}

		// 2. Если «просмотровой» сессии нет — выбрать стрим и открыть её.
		if variant == "" {
			streams, err := w.gql.TopStreams(ctx, w.m.cfg.TargetGame, 15)
			if err != nil {
				return err
			}
			if len(streams) == 0 {
				w.st.addLog("Twitch не вернул живые стримы с включёнными Drops для категории «" + w.m.cfg.TargetGame + "», повтор через паузу")
				if !sleepCtx(ctx, jittered(rnd, 60*time.Second, w.m.cfg.JitterSec)) {
					return nil
				}
				continue
			}
			s := streams[rnd.Intn(len(streams))] // случайный выбор — разносим аккаунты по каналам
			token, sig, err := w.gql.PlaybackAccessToken(ctx, s.Login)
			if err != nil {
				return fmt.Errorf("playback token: %w", err)
			}
			master, err := w.gql.GetMasterPlaylist(ctx, twitch.MasterPlaylistURL(s.Login, token, sig))
			if err != nil {
				return fmt.Errorf("master playlist: %w", err)
			}
			variant, err = twitch.PickLowestVariant(master)
			if err != nil {
				return err
			}
			watchStream = s
			w.st.addLog("просмотр канала: " + watchStream.Login)
			continue // первый heartbeat — на следующей итерации
		}

		// 3. Ждём до фиксированного минутного дедлайна. Время сетевого
		// запроса не прибавляется к интервалу и не сдвигает все следующие циклы.
		if !nextHeartbeat.IsZero() && !sleepUntil(ctx, nextHeartbeat) {
			return nil
		}

		// 4. Периодический опрос прогресса и клейм.
		if time.Since(lastPoll) < pollInterval {
			continue
		}
		lastPoll = time.Now()
		cur, ok, err := w.findDrop(ctx, drop.ID)
		if err != nil {
			w.st.addLog("прогресс: " + err.Error())
			continue
		}
		if !ok {
			w.st.addLog("дроп пропал из дашборда")
			return nil
		}
		if cur.IsClaimed {
			w.st.addLog("«" + cur.Name + "» уже получен")
			return nil
		}
		if cur.Current != drop.Current {
			w.st.addLog(fmt.Sprintf("прогресс «%s»: %d/%d мин", cur.Name, cur.Current, cur.Required))
			drop = cur
		}
		if cur.Current >= cur.Required {
			if cur.DropInstanceID != "" {
				return w.claim(ctx, cur)
			}
			pendingClaim++
			w.st.addLog("100% набрано, ожидаю dropInstanceID…")
			if pendingClaim >= 15 {
				w.st.addLog("dropInstanceID не выдан — пропускаю дроп")
				return nil
			}
		}
	}
	return nil
}

// claim — автоматический GQL-клейм готового дропа.
func (w *accountWorker) claim(ctx context.Context, drop twitch.Drop) error {
	if err := w.gql.ClaimDrop(ctx, drop.DropInstanceID); err != nil {
		return fmt.Errorf("claim: %w", err)
	}
	w.st.mu.Lock()
	w.st.DropsClaimed++
	w.st.mu.Unlock()
	w.st.addLog("✅ ДРОП ПОЛУЧЕН: " + drop.Name)
	w.m.notifyClaim(w.acc, drop)
	return nil
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

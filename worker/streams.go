package worker

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/xyloni/tdm-light/twitch"
)

const channelCooldown = 5 * time.Minute

func (w *accountWorker) excludeChannel(login string) {
	if w.unavailable == nil {
		w.unavailable = make(map[string]time.Time)
	}
	// Remove expired entries, keeping per-account state bounded over long runs.
	now := time.Now()
	for channel, until := range w.unavailable {
		if !now.Before(until) {
			delete(w.unavailable, channel)
		}
	}
	w.unavailable[login] = now.Add(channelCooldown)
}

func (w *accountWorker) openStream(ctx context.Context, rnd *rand.Rand) (twitch.Stream, string, error) {
	return w.openStreamExcept(ctx, rnd, "")
}

func (w *accountWorker) openChannel(ctx context.Context, s twitch.Stream) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, channelOpenBudget)
	defer cancel()
	token, sig, err := w.gql.PlaybackAccessToken(ctx, s.Login)
	if err != nil {
		return "", err
	}
	master, err := w.gql.GetMasterPlaylist(ctx, twitch.MasterPlaylistURL(s.Login, token, sig))
	if err != nil {
		return "", err
	}
	return twitch.PickLowestVariant(master)
}

func (w *accountWorker) recoverSlowStream(ctx context.Context, rnd *rand.Rand, current twitch.Stream) (twitch.Stream, string, error) {
	ctx, cancel := context.WithTimeout(ctx, streamRecoveryBudget)
	defer cancel()
	w.excludeChannel(current.Login)
	selected, playlist, err := w.openStreamExcept(ctx, rnd, current.Login)
	if err == nil {
		w.st.addLog("смена канала из-за медленного начисления: " + current.Login + " → " + selected.Login)
		return selected, playlist, nil
	}
	if ctx.Err() != nil || isAuthError(err) {
		return twitch.Stream{}, "", err
	}
	playlist, err = w.openChannel(ctx, current)
	if err != nil {
		return twitch.Stream{}, "", err
	}
	delete(w.unavailable, current.Login)
	w.st.addLog("замены каналу " + current.Login + " нет; текущий просмотр переподключён")
	return current, playlist, nil
}

func (w *accountWorker) openStreamExcept(ctx context.Context, rnd *rand.Rand, excluded string) (twitch.Stream, string, error) {
	ctx, cancel := context.WithTimeout(ctx, streamRecoveryBudget)
	defer cancel()
	streams, err := w.gql.TopStreams(ctx, w.m.cfg.TargetGame, 15)
	if err != nil {
		return twitch.Stream{}, "", err
	}
	attempts := 0
	for _, i := range rnd.Perm(len(streams)) {
		s := streams[i]
		if s.Login == excluded {
			continue
		}
		if time.Now().Before(w.unavailable[s.Login]) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return twitch.Stream{}, "", err
		}
		attempts++
		variant, err := w.openChannel(ctx, s)
		if err == nil {
			return s, variant, nil
		}
		if ctx.Err() != nil {
			return twitch.Stream{}, "", ctx.Err()
		}
		if isAuthError(err) {
			return twitch.Stream{}, "", err
		}
		w.excludeChannel(s.Login)
		// Do not print signed playlist URLs from transport errors.
		w.st.addLog("не удалось открыть канал " + s.Login + "; исключён на 5 минут")
		if attempts >= 3 {
			break // Bound recovery work before the worker's normal retry pause.
		}
	}
	return twitch.Stream{}, "", fmt.Errorf("нет доступных каналов вне временного исключения (проверено %d); повтор после паузы", attempts)
}

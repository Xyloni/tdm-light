package worker

import (
	"math/rand"
	"time"
)

const (
	rateWindow           = 10 * time.Minute
	rateMaxWindow        = 15 * time.Minute
	rateRecoveryCooldown = 10 * time.Minute
)

type progressSample struct {
	at      time.Time
	minutes int
}
type watchRate struct{ samples []progressSample }

type rateSnapshot struct {
	gain    int
	elapsed time.Duration
	checked time.Time
	ready   bool
}

func (s *AccountState) setWatchRate(r *watchRate) {
	snapshot := rateSnapshot{}
	if len(r.samples) >= 4 {
		first, last := r.samples[0], r.samples[len(r.samples)-1]
		elapsed := last.at.Sub(first.at)
		if elapsed >= rateWindow && elapsed <= rateMaxWindow {
			snapshot = rateSnapshot{gain: last.minutes - first.minutes, elapsed: elapsed, checked: last.at, ready: true}
		}
	}
	s.mu.Lock()
	s.rateSnapshot = snapshot
	s.mu.Unlock()
}

func (s *AccountState) clearWatchRate() {
	s.mu.Lock()
	s.rateSnapshot = rateSnapshot{}
	s.mu.Unlock()
}

func (r *watchRate) reset(at time.Time, minutes int, known bool) {
	r.samples = r.samples[:0]
	if known {
		r.samples = append(r.samples, progressSample{at, minutes})
	}
}

// A small positive increment does not reset the observation window. Unknown
// progress, a backwards clock/counter, or a long observation gap does.
func (r *watchRate) observe(at time.Time, minutes int, known bool, maxGap time.Duration) (bool, int, time.Duration) {
	if !known {
		r.reset(at, minutes, false)
		return false, 0, 0
	}
	if len(r.samples) > 0 {
		last := r.samples[len(r.samples)-1]
		if !at.After(last.at) || minutes < last.minutes || at.Sub(last.at) > maxGap {
			r.reset(at, minutes, true)
			return false, 0, 0
		}
	}
	r.samples = append(r.samples, progressSample{at, minutes})
	for len(r.samples) > 1 && at.Sub(r.samples[1].at) >= rateWindow {
		r.samples = r.samples[1:]
	}
	first := r.samples[0]
	elapsed, gain := at.Sub(first.at), minutes-first.minutes
	if elapsed > rateMaxWindow {
		r.reset(at, minutes, true)
		return false, 0, 0
	}
	// One minute allowance for integer rounding and reporting lag.
	slow := len(r.samples) >= 4 && elapsed >= rateWindow && gain+1 < int(elapsed.Minutes()*0.8)
	return slow, gain, elapsed
}

func (w *accountWorker) reserveRateRecovery(now time.Time, rnd *rand.Rand) bool {
	if now.Before(w.nextRateRecovery) {
		return false
	}
	w.nextRateRecovery = now.Add(rateRecoveryCooldown + time.Duration(rnd.Intn(121))*time.Second)
	return true
}

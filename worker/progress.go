package worker

import (
	"fmt"
	"time"

	"github.com/xyloni/tdm-light/twitch"
)

func (s *AccountState) setProgress(drop twitch.Drop) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentDrop = drop.Name
	s.CurrentMinutes = drop.Current
	s.RequiredMinutes = drop.Required
	s.ProgressKnown = drop.ProgressKnown
	s.ProgressChecked = time.Now()
}

func (w *accountWorker) reportProgress(drop twitch.Drop) {
	w.st.setProgress(drop)
	if !drop.ProgressKnown {
		w.st.addLog("прогресс «" + drop.Name + "»: личные минуты пока не получены из Twitch Inventory")
		return
	}
	w.st.addLog(fmt.Sprintf("прогресс «%s»: %d/%d мин (Twitch Inventory)", drop.Name, drop.Current, drop.Required))
}

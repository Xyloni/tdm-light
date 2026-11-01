package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xyloni/tdm-light/bot"
	"github.com/xyloni/tdm-light/config"
	"github.com/xyloni/tdm-light/parser"
	"github.com/xyloni/tdm-light/twitch"
)

const maxLogLines = 100 // глубина in-memory буфера логов на аккаунт

// AccountState — состояние и кольцевой буфер логов одного аккаунта.
type AccountState struct {
	Account         *parser.Account
	LogFile         string
	mu              sync.Mutex
	Status          string
	DropsReady      int
	readyDrops      map[string]struct{}
	authFailed      atomic.Bool
	logs            []string
	logNext         int
	fileMu          sync.Mutex
	CurrentDrop     string
	CurrentMinutes  int
	RequiredMinutes int
	ProgressKnown   bool
	ProgressChecked time.Time
	rateSnapshot    rateSnapshot
	rateRecoveries  int
}

func (s *AccountState) setStatus(v string) {
	s.mu.Lock()
	s.Status = v
	s.mu.Unlock()
}

func (s *AccountState) addLog(line string) {
	stamped := time.Now().Format("15:04:05") + " " + line
	s.mu.Lock()
	if len(s.logs) < maxLogLines {
		s.logs = append(s.logs, stamped)
	} else {
		s.logs[s.logNext] = stamped
	}
	s.logNext = (s.logNext + 1) % maxLogLines
	s.mu.Unlock()
	log.Printf("[%s] %s", s.Account.Login, line)
	if s.LogFile != "" {
		if err := s.appendFileLog(time.Now().Format("2006-01-02 ") + stamped + "\n"); err != nil {
			log.Printf("[%s] запись лога: %v", s.Account.Login, err)
		}
	}
}

func (s *AccountState) tailLogs(n int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 {
		return nil
	}
	if n > len(s.logs) {
		n = len(s.logs)
	}
	start := len(s.logs) - n
	if len(s.logs) == maxLogLines {
		start = (s.logNext + start) % maxLogLines
	}
	out := make([]string, n)
	for i := range out {
		out[i] = s.logs[(start+i)%len(s.logs)]
	}
	return out
}

// Manager управляет группами аккаунтов: фарм идёт поочерёдно группами,
// внутри группы каждый аккаунт — отдельная горутина.
type Manager struct {
	cfg      *config.Config
	groups   [][]*parser.Account
	states   map[string]*AccountState // ключ — login (логины должны быть уникальны)
	http     *http.Client
	bot      *bot.Bot
	reloadCh chan struct{}

	started      time.Time
	totalReady   atomic.Int64
	currentGroup atomic.Int32
}

func NewManager(cfg *config.Config, groups [][]*parser.Account, reloadCh chan struct{}, states ...*bot.PollState) *Manager {
	m := &Manager{
		cfg:      cfg,
		groups:   groups,
		states:   make(map[string]*AccountState),
		reloadCh: reloadCh,
		started:  time.Now(),
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        128,
				MaxIdleConnsPerHost: 32,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
	if cfg.LogDir != "" {
		_ = os.MkdirAll(cfg.LogDir, 0o755)
	}
	for _, g := range groups {
		for _, acc := range g {
			logFile := ""
			if cfg.LogDir != "" {
				logFile = filepath.Join(cfg.LogDir, acc.Login+".log")
			}
			m.states[acc.Login] = &AccountState{Account: acc, Status: "init", LogFile: logFile}
		}
	}
	if cfg.TelegramBotToken != "" && cfg.TelegramAdminID != 0 {
		m.bot = bot.New(cfg.TelegramBotToken, cfg.TelegramAdminID,
			m.statusText, m.logsText, func() {
				select {
				case m.reloadCh <- struct{}{}:
				default:
				}
			}, func() string { return m.cfg.TargetGame }, func(game string) error {
				return saveTargetGame("config.env", game)
			}, func(content []byte) error {
				return replaceAccountsFile(cfg.AccountsFile, content)
			}, states...)
	} else {
		log.Println("telegram-бот отключён (не заданы telegram_bot_token / telegram_admin_id)")
	}
	return m
}

// Run запускает бота и цикл поочерёдного фарма групп.
// Возвращает true, если запрошена горячая перезагрузка (команда /restart).
func (m *Manager) Run(parent context.Context) (reloaded bool) {
	ctx, cancel := context.WithCancel(parent)
	botDone := make(chan struct{})
	defer func() {
		cancel()
		<-botDone
		m.http.CloseIdleConnections()
	}()

	if m.bot != nil {
		go func() {
			defer close(botDone)
			m.bot.Run(ctx)
		}()
	} else {
		close(botDone)
	}
	total := 0
	for _, g := range m.groups {
		total += len(g)
	}
	log.Printf("загружено аккаунтов: %d, групп: %d, целевая игра: %q", total, len(m.groups), m.cfg.TargetGame)
	if total == 0 {
		log.Println("майнер ожидает account.txt; отправьте файл боту или добавьте его на сервер и выполните /restart")
		select {
		case <-m.reloadCh:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		cycleStarted := time.Now()
		for gi, group := range m.groups {
			if parent.Err() != nil {
				return false
			}
			m.currentGroup.Store(int32(gi + 1))
			if m.runGroup(ctx, group) {
				return true // reload
			}
		}
		// Completed or unauthorized groups must not cause an immediate retry loop.
		timer := time.NewTimer(time.Until(cycleStarted.Add(time.Duration(maxInt(m.cfg.CampaignPollIntervalSec, 60)) * time.Second)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-m.reloadCh:
			timer.Stop()
			return true
		case <-timer.C:
		}
	}
}

// runGroup фармит одну группу: все аккаунты группы параллельно,
// сессия ограничена group_session_minutes (0 — пока есть что фармить).
// Возвращает true при запросе перезагрузки.
func (m *Manager) runGroup(ctx context.Context, group []*parser.Account) (reloaded bool) {
	if len(group) == 0 {
		return false
	}
	var sessCtx context.Context
	var cancel context.CancelFunc
	if m.cfg.GroupSessionMinutes > 0 {
		sessCtx, cancel = context.WithTimeout(ctx, time.Duration(m.cfg.GroupSessionMinutes)*time.Minute)
	} else {
		sessCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var wg sync.WaitGroup
	groupSize := len(group)
	for i, acc := range group {
		st := m.states[acc.Login]
		if st.authFailed.Load() {
			continue // Authorization is retried only after an explicit reload.
		}
		// Равномерно распределяем первичный запуск аккаунтов по первой минуте.
		// После этой задержки каждый воркер остаётся активным и продолжает
		// собственный цикл heartbeat без повторного стартового ожидания.
		startDelay := startupDelay(i, groupSize)
		w := &accountWorker{
			m: m, acc: acc, st: st,
			gql:          twitch.NewGQLClient(m.http, m.cfg, acc),
			startupDelay: startDelay,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Run(sessCtx)
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done: // группа отработала свою сессию
		return false
	case <-m.reloadCh: // /restart
		cancel()
		<-done // ждём корректного завершения всех горутин группы
		return true
	case <-ctx.Done(): // SIGINT/SIGTERM
		cancel()
		<-done
		return false
	}
}

// --- сводки для Telegram-бота ---

func (m *Manager) logsText(login string) string {
	st, ok := m.states[strings.ToLower(login)]
	if !ok {
		return "Аккаунт не найден: " + login
	}
	st.mu.Lock()
	progress := ""
	if st.CurrentDrop != "" {
		if st.ProgressKnown {
			progress = fmt.Sprintf("🎯 %s: %d/%d мин (Twitch Inventory, проверено %s)\n", st.CurrentDrop, st.CurrentMinutes, st.RequiredMinutes, st.ProgressChecked.Format("15:04:05"))
		} else {
			progress = "🎯 " + st.CurrentDrop + ": личный прогресс пока не получен из Twitch Inventory\n"
		}
	}
	st.mu.Unlock()
	lines := st.tailLogs(50)
	if len(lines) == 0 {
		return "Логов пока нет."
	}
	return progress + strings.Join(lines, "\n")
}

func (m *Manager) notifyReady(acc *parser.Account, drop twitch.Drop) {
	m.totalReady.Add(1)
	if m.bot != nil {
		m.bot.Notify(fmt.Sprintf("🎁 Время для дропа набрано! Получение вручную.\nАккаунт: %s\nДроп: %s\nКампания: %s",
			acc.Login, drop.Name, drop.CampaignName))
	}
}

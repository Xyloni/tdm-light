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
	Account      *parser.Account
	LogFile      string
	mu           sync.Mutex
	Status       string
	DropsClaimed int
	logs         []string
}

func (s *AccountState) setStatus(v string) {
	s.mu.Lock()
	s.Status = v
	s.mu.Unlock()
}

func (s *AccountState) addLog(line string) {
	stamped := time.Now().Format("15:04:05") + " " + line
	s.mu.Lock()
	s.logs = append(s.logs, stamped)
	if len(s.logs) > maxLogLines {
		s.logs = append([]string(nil), s.logs[len(s.logs)-maxLogLines:]...)
	}
	s.mu.Unlock()
	log.Printf("[%s] %s", s.Account.Login, line)
	if s.LogFile != "" {
		if f, err := os.OpenFile(s.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintln(f, time.Now().Format("2006-01-02 ")+stamped)
			f.Close()
		}
	}
}

func (s *AccountState) tailLogs(n int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logs) > n {
		return append([]string(nil), s.logs[len(s.logs)-n:]...)
	}
	return append([]string(nil), s.logs...)
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
	totalClaims  atomic.Int64
	currentGroup atomic.Int32
}

func NewManager(cfg *config.Config, groups [][]*parser.Account, reloadCh chan struct{}) *Manager {
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
				return updateEnvSetting("config.env", "TARGET_GAME", game)
			}, func(content []byte) error {
				return replaceAccountsFile(cfg.AccountsFile, content)
			})
	} else {
		log.Println("telegram-бот отключён (не заданы telegram_bot_token / telegram_admin_id)")
	}
	return m
}

// Run запускает бота и цикл поочерёдного фарма групп.
// Возвращает true, если запрошена горячая перезагрузка (команда /restart).
func (m *Manager) Run(parent context.Context) (reloaded bool) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	if m.bot != nil {
		go m.bot.Run(ctx)
	}
	total := 0
	for _, g := range m.groups {
		total += len(g)
	}
	log.Printf("загружено аккаунтов: %d, групп: %d, целевая игра: %q", total, len(m.groups), m.cfg.TargetGame)
	if total == 0 {
		log.Println("майнер ожидает accounts.txt; отправьте файл боту или добавьте его на сервер и выполните /restart")
		select {
		case <-m.reloadCh:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		for gi, group := range m.groups {
			if parent.Err() != nil {
				return false
			}
			m.currentGroup.Store(int32(gi + 1))
			if m.runGroup(ctx, group) {
				return true // reload
			}
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

func (m *Manager) statusText() string {
	var farming, idle, errs int
	for _, st := range m.states {
		st.mu.Lock()
		s := st.Status
		st.mu.Unlock()
		switch {
		case strings.HasPrefix(s, "farm"):
			farming++
		case strings.HasPrefix(s, "error"):
			errs++
		default:
			idle++
		}
	}
	return fmt.Sprintf(
		"🎮 Игра: %s\n👥 Аккаунтов: %d (фармят: %d · простой: %d · ошибок: %d)\n🔢 Группа: %d/%d\n🎁 Дропов получено: %d\n⏱ Аптайм: %s",
		m.cfg.TargetGame, len(m.states), farming, idle, errs,
		m.currentGroup.Load(), len(m.groups),
		m.totalClaims.Load(), time.Since(m.started).Round(time.Second),
	)
}

func (m *Manager) logsText(login string) string {
	st, ok := m.states[login]
	if !ok {
		return "Аккаунт не найден: " + login
	}
	lines := st.tailLogs(50)
	if len(lines) == 0 {
		return "Логов пока нет."
	}
	return strings.Join(lines, "\n")
}

func (m *Manager) notifyClaim(acc *parser.Account, drop twitch.Drop) {
	m.totalClaims.Add(1)
	if m.bot != nil {
		m.bot.Notify(fmt.Sprintf("🎁 Дроп получен!\nАккаунт: %s\nДроп: %s\nКампания: %s",
			acc.Login, drop.Name, drop.CampaignName))
	}
}

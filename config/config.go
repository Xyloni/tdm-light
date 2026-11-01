package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config — вся конфигурация сервиса.
type Config struct {
	TargetGame              string `json:"target_game"`                // ключевая игра для фарма
	ClientID                string `json:"client_id"`                  // Client-ID клиента, которым выданы OAuth-токены
	TelegramBotToken        string `json:"telegram_bot_token"`         // токен TG-бота
	TelegramAdminID         int64  `json:"telegram_admin_id"`          // ID администратора (chat_id)
	AccountsFile            string `json:"accounts_file"`              // путь к account.txt
	HeartbeatIntervalSec    int    `json:"heartbeat_interval_sec"`     // интервал beacon, минимум 60 секунд; HLS отдельно каждые 20 секунд
	CampaignPollIntervalSec int    `json:"campaign_poll_interval_sec"` // интервал опроса прогресса дропов
	JitterSec               int    `json:"jitter_sec"`                 // ± секунд случайного разброса
	GroupSessionMinutes     int    `json:"group_session_minutes"`      // длительность сессии группы (0 = до конца кампаний)
	LogDir                  string `json:"log_dir"`                    // каталог текстовых логов аккаунтов
	UsePubSub               bool   `json:"use_pubsub"`                 // опциональный WebSocket PubSub для событий дропов
}

// Default возвращает конфигурацию с безопасными значениями по умолчанию.
func Default() *Config {
	return &Config{
		ClientID:                "kd1unb4b3q4t58fwlpcbzcbnm76a8fp", // Client-ID приложения Twitch для Android
		AccountsFile:            "account.txt",
		HeartbeatIntervalSec:    60,
		CampaignPollIntervalSec: 120,
		JitterSec:               20,
		GroupSessionMinutes:     60,
		LogDir:                  "logs",
	}
}

// Load читает config.json (если существует), затем применяет переопределения
// из config.env и переменных окружения (приоритет: env > config.env > config.json).
func Load(jsonPath, envPath string) (*Config, error) {
	cfg := Default()

	if data, err := os.ReadFile(jsonPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("%s: %w", jsonPath, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("%s: %w", jsonPath, err)
	}

	applyEnvFile(cfg, envPath)
	applyOSEnv(cfg)
	// Normalize the previous conventional filename, including absolute Docker paths.
	if filepath.Base(cfg.AccountsFile) == "accounts.txt" {
		cfg.AccountsFile = filepath.Join(filepath.Dir(cfg.AccountsFile), "account.txt")
	}

	if strings.TrimSpace(cfg.TargetGame) == "" {
		return nil, fmt.Errorf("target_game не задан ни в config.json, ни в окружении")
	}
	return cfg, nil
}

// applyEnvFile разбирает простой KEY=VALUE файл.
func applyEnvFile(cfg *Config, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		applyKV(cfg, strings.TrimSpace(k), strings.TrimSpace(v))
	}
}

func applyOSEnv(cfg *Config) {
	for _, k := range []string{
		"TARGET_GAME", "CLIENT_ID", "TELEGRAM_BOT_TOKEN", "TELEGRAM_ADMIN_ID",
		"ACCOUNTS_FILE", "HEARTBEAT_INTERVAL_SEC", "CAMPAIGN_POLL_INTERVAL_SEC",
		"JITTER_SEC", "GROUP_SESSION_MINUTES", "LOG_DIR", "USE_PUBSUB",
	} {
		if v, ok := os.LookupEnv(k); ok {
			applyKV(cfg, k, v)
		}
	}
}

func applyKV(cfg *Config, key, val string) {
	switch strings.ToUpper(key) {
	case "TARGET_GAME":
		cfg.TargetGame = val
	case "CLIENT_ID":
		cfg.ClientID = val
	case "TELEGRAM_BOT_TOKEN":
		cfg.TelegramBotToken = val
	case "TELEGRAM_ADMIN_ID":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.TelegramAdminID = n
		}
	case "ACCOUNTS_FILE":
		cfg.AccountsFile = val
	case "HEARTBEAT_INTERVAL_SEC":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.HeartbeatIntervalSec = n
		}
	case "CAMPAIGN_POLL_INTERVAL_SEC":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.CampaignPollIntervalSec = n
		}
	case "JITTER_SEC":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.JitterSec = n
		}
	case "GROUP_SESSION_MINUTES":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.GroupSessionMinutes = n
		}
	case "LOG_DIR":
		cfg.LogDir = val
	case "USE_PUBSUB":
		cfg.UsePubSub = val == "1" || strings.EqualFold(val, "true")
	}
}

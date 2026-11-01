package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

const maxAccountsUploadBytes = 2 << 20

type telegramDocument struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// PollState survives manager reloads. A process owns one state for its bot.
type PollState struct {
	offset atomic.Int64
}

// Bot — минимальный Telegram-бот на long polling.
// Команды принимаются только от администратора (adminID).
type Bot struct {
	token     string
	adminID   int64
	http      *http.Client
	pollState *PollState

	statusFn func() string
	logsFn   func(login string) string
	reloadFn func()
	gameFn   func() string
	setGame  func(string) error
	uploadFn func([]byte) error
}

func New(token string, adminID int64, statusFn func() string, logsFn func(string) string, reloadFn func(), gameFn func() string, setGame func(string) error, uploadFn func([]byte) error, states ...*PollState) *Bot {
	state := new(PollState)
	if len(states) > 0 && states[0] != nil {
		state = states[0]
	}
	return &Bot{
		pollState: state,
		token:     token,
		adminID:   adminID,
		http:      &http.Client{Timeout: 40 * time.Second},
		statusFn:  statusFn,
		logsFn:    logsFn,
		reloadFn:  reloadFn,
		gameFn:    gameFn,
		setGame:   setGame,
		uploadFn:  uploadFn,
	}
}

// Run крутит long polling до отмены ctx.
func (b *Bot) Run(ctx context.Context) {
	log.Println("[bot] long polling запущен")
	for ctx.Err() == nil {
		if err := b.poll(ctx); err != nil && ctx.Err() == nil {
			log.Printf("[bot] getUpdates: %v", err)
			if !sleepCtx(ctx, 5*time.Second) {
				return
			}
		}
	}
}

func (b *Bot) poll(ctx context.Context) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=25&allowed_updates=%%5B%%22message%%22%%5D",
		b.token, b.pollState.offset.Load())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res struct {
		OK     bool `json:"ok"`
		Result []struct {
			UpdateID int64 `json:"update_id"`
			Message  *struct {
				Text     string            `json:"text"`
				Document *telegramDocument `json:"document"`
				From     struct {
					ID int64 `json:"id"`
				} `json:"from"`
			} `json:"message"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.OK {
		return fmt.Errorf("telegram: ok=false")
	}
	for _, upd := range res.Result {
		if err := ctx.Err(); err != nil {
			return err
		}
		if upd.UpdateID < b.pollState.offset.Load() {
			continue
		}
		b.pollState.offset.Store(upd.UpdateID + 1)
		msg := upd.Message
		if msg == nil || msg.From.ID != b.adminID {
			continue
		}
		b.handle(ctx, msg.Text, msg.Document)
	}
	return nil
}

func (b *Bot) handle(ctx context.Context, text string, document *telegramDocument) {
	if document != nil {
		b.handleAccountsDocument(ctx, document.FileID, document.FileName, document.FileSize)
		return
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return
	}
	cmd := strings.ToLower(strings.SplitN(fields[0], "@", 2)[0])
	switch cmd {
	case "/help", "/start":
		b.send(helpText)
	case "/status":
		b.send(b.statusFn())
	case "/logs":
		if len(fields) < 2 {
			b.send("Использование: /logs <login>")
			return
		}
		b.send(b.logsFn(fields[1]))
	case "/game":
		game := ""
		if len(fields) > 1 {
			game = strings.TrimSpace(text[len(fields[0]):])
		}
		if game == "" {
			current := "не задана"
			if b.gameFn != nil {
				if configured := b.gameFn(); configured != "" {
					current = configured
				}
			}
			b.send("Текущая игра: " + current + "\nИзменить: /game Название игры")
			return
		}
		if b.setGame == nil {
			b.send("Изменение игры не настроено.")
			return
		}
		if err := b.setGame(game); err != nil {
			b.send("Не удалось сохранить игру: " + err.Error())
			return
		}
		b.send("🎮 Игра сохранена: " + game + "\nПерезапускаю майнинг с новой игрой.")
		if b.reloadFn != nil {
			b.reloadFn()
		}
	case "/restart":
		b.send("♻️ Перезагружаю config.json и account.txt...")
		b.reloadFn()
	default:
		b.send(helpText)
	}
}

const helpText = "Команды:\n/status — сводка фарма\n/logs <login> — последние логи аккаунта\n/game — показать текущую игру\n/game <название> — сменить игру и перезапустить майнинг\n/restart — перечитать конфиг и account.txt\n\nЧтобы загрузить аккаунты, отправьте любой .txt документом. Файл будет проверен, сохранён как account.txt и майнинг перезапустится."

func (b *Bot) handleAccountsDocument(ctx context.Context, fileID, fileName string, fileSize int64) {
	if !strings.EqualFold(path.Ext(fileName), ".txt") || strings.ContainsAny(fileName, `/\\`) {
		b.send("Пришли документ с любым названием и расширением .txt.")
		return
	}
	if b.uploadFn == nil {
		b.send("Загрузка account.txt не настроена.")
		return
	}
	if fileSize > maxAccountsUploadBytes {
		b.send("Файл слишком большой (максимум 2 МБ).")
		return
	}
	content, err := b.downloadDocument(ctx, fileID)
	if err != nil {
		b.send("Не удалось скачать файл из Telegram: " + err.Error())
		return
	}
	if err := b.uploadFn(content); err != nil {
		b.send("Файл не сохранён: " + err.Error())
		return
	}
	b.send("✅ account.txt проверен и сохранён. Перезапускаю фарм с новым списком аккаунтов.")
	if b.reloadFn != nil {
		b.reloadFn()
	}
}

func (b *Bot) downloadDocument(ctx context.Context, fileID string) ([]byte, error) {
	payload, _ := json.Marshal(map[string]string{"file_id": fileID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.telegram.org/bot%s/getFile", b.token), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getFile вернул HTTP %d", resp.StatusCode)
	}
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK || result.Result.FilePath == "" {
		return nil, fmt.Errorf("Telegram не вернул путь к файлу")
	}
	cleanPath := path.Clean(result.Result.FilePath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") || strings.HasPrefix(cleanPath, "/") {
		return nil, fmt.Errorf("Telegram вернул некорректный путь к файлу")
	}
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.token, cleanPath)
	fileReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}
	fileResp, err := b.http.Do(fileReq)
	if err != nil {
		return nil, err
	}
	defer fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("скачивание вернуло HTTP %d", fileResp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(fileResp.Body, maxAccountsUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxAccountsUploadBytes {
		return nil, fmt.Errorf("файл слишком большой (максимум 2 МБ)")
	}
	return content, nil
}

// Notify отправляет уведомление администратору (например, о полученном дропе).
func (b *Bot) Notify(text string) { b.send(text) }

func (b *Bot) send(text string) {
	if len(text) > 4000 { // лимит Telegram — 4096 символов
		text = text[:4000] + "…"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]any{
		"chat_id": b.adminID,
		"text":    text,
	})
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		log.Printf("[bot] sendMessage: %v", err)
		return
	}
	resp.Body.Close()
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

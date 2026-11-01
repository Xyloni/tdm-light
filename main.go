package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xyloni/tdm-light/bot"
	"github.com/xyloni/tdm-light/config"
	"github.com/xyloni/tdm-light/parser"
	"github.com/xyloni/tdm-light/worker"
)

func main() {
	log.SetPrefix("[tdm-light] ")
	log.SetFlags(log.LstdFlags)

	// Graceful shutdown по SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Канал горячей перезагрузки (команда /restart из Telegram).
	reloadCh := make(chan struct{}, 1)
	pollState := new(bot.PollState)
	previousBotToken := ""

	for {
		cfg, err := config.Load("config.json", "config.env")
		if err != nil {
			log.Fatalf("конфигурация: %v", err)
		}
		if err := parser.MigrateLegacyFile(cfg.AccountsFile); err != nil {
			log.Fatalf("перенос файла аккаунтов: %v", err)
		}
		groups, err := parser.ParseFile(cfg.AccountsFile)
		if err != nil {
			if errors.Is(err, parser.ErrNoAccounts) || os.IsNotExist(err) {
				log.Printf("список аккаунтов пуст или отсутствует (%s); можно отправить account.txt Telegram-боту", cfg.AccountsFile)
				groups = nil
			} else {
				log.Fatalf("аккаунты: %v", err)
			}
		}

		if cfg.TelegramBotToken != previousBotToken {
			pollState = new(bot.PollState)
			previousBotToken = cfg.TelegramBotToken
		}
		mgr := worker.NewManager(cfg, groups, reloadCh, pollState)
		if !mgr.Run(ctx) {
			break // shutdown
		}
		log.Println("горячая перезагрузка config.json и account.txt...")
	}
	log.Println("сервис остановлен")
}

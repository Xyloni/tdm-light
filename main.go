package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dropfarm/config"
	"dropfarm/parser"
	"dropfarm/worker"
)

func main() {
	log.SetPrefix("[dropfarm] ")
	log.SetFlags(log.LstdFlags)

	// Graceful shutdown по SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Канал горячей перезагрузки (команда /restart из Telegram).
	reloadCh := make(chan struct{}, 1)

	for {
		cfg, err := config.Load("config.json", "config.env")
		if err != nil {
			log.Fatalf("конфигурация: %v", err)
		}
		groups, err := parser.ParseFile(cfg.AccountsFile)
		if err != nil {
			if errors.Is(err, parser.ErrNoAccounts) || os.IsNotExist(err) {
				log.Printf("список аккаунтов пуст или отсутствует (%s); можно отправить accounts.txt Telegram-боту", cfg.AccountsFile)
				groups = nil
			} else {
				log.Fatalf("аккаунты: %v", err)
			}
		}

		mgr := worker.NewManager(cfg, groups, reloadCh)
		if !mgr.Run(ctx) {
			break // shutdown
		}
		log.Println("горячая перезагрузка config.json и accounts.txt...")
	}
	log.Println("сервис остановлен")
}

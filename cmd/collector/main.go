package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adda-Baaj/taja-khobor/internal/app"
	"github.com/Adda-Baaj/taja-khobor/internal/config"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "collector start failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.Init(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Close()

	logger.InfoObj("collector starting", "config", cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	collector, err := app.NewCollector(cfg, log)
	if err != nil {
		logger.ErrorObj("failed to initialize collector", "error", err)
		return err
	}

	if err := collector.Run(ctx); err != nil {
		return fmt.Errorf("collector run: %w", err)
	}

	return nil
}

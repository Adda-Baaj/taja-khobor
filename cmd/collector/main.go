package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Adda-Baaj/taja-khobor/internal/config"
	"github.com/Adda-Baaj/taja-khobor/internal/crawler"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
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

	_, err = logger.Init(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Close()

	logger.InfoObj("collector starting", "config", cfg)

	if err := providers.LoadProviders(cfg.ProvidersFile); err != nil {
		logger.ErrorObj("failed to load providers registry", "error", err)
		return fmt.Errorf("load providers registry: %w", err)
	}
	resolvedProviders := providers.Providers()
	logger.InfoObj("providers registry loaded", "providers", resolvedProviders)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if len(resolvedProviders) == 0 {
		logger.WarnObj("no providers configured; collector exiting", "providers_file", cfg.ProvidersFile)
		return nil
	}

	crawlService := crawler.NewService(providers.DefaultFetcherRegistry(nil))
	if err := crawlService.Run(ctx, resolvedProviders); err != nil {
		logger.ErrorObj("crawl completed with errors", "error", err)
		return fmt.Errorf("crawl execution: %w", err)
	}

	logger.InfoObj("collector waiting for shutdown signal", "providers_count", len(resolvedProviders))
	<-ctx.Done()

	logger.InfoObj("collector shutting down gracefully", "reason", ctx.Err())

	return nil
}

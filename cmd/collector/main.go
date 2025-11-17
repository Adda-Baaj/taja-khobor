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

// Entrypoint for the collector service. Wire up config, logger, scheduler, etc. here.
func main() {
	fmt.Println("collector starting")

	// load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	// initialize logger
	_, err = logger.Init(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.InfoObj("collector started", "config", cfg)

	// load providers registry and log the entries we found for traceability
	if err := providers.LoadProviders(cfg.ProvidersFile); err != nil {
		fmt.Fprintf(os.Stderr, "load providers: %v\n", err)
		os.Exit(1)
	}
	logger.InfoObj("providers registry loaded", "providers", providers.Providers())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	resolvedProviders := providers.Providers()
	if len(resolvedProviders) == 0 {
		logger.WarnObj("no providers configured; collector exiting", "providers_file", cfg.ProvidersFile)
		return
	}

	crawlService := crawler.NewService(providers.DefaultFetcherRegistry(nil))
	if err := crawlService.Run(ctx, resolvedProviders); err != nil {
		logger.ErrorObj("crawl completed with errors", "error", err)
	}

	logger.InfoObj("collector waiting for shutdown signal", "providers_count", len(resolvedProviders))
	<-ctx.Done()
	logger.InfoObj("collector shutting down gracefully", "reason", ctx.Err())
}

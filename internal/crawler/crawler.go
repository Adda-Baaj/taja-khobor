package crawler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
)

const maxProviderWorkers = 10

type Service struct {
	registry providers.FetcherRegistry
	scraper  ArticleScraper
}

func NewService(reg providers.FetcherRegistry) *Service {
	return &Service{
		registry: reg,
		scraper:  NewScraper(nil),
	}
}

func (s *Service) Run(ctx context.Context, cfgs []providers.Provider) error {
	if s == nil || s.registry == nil {
		return fmt.Errorf("crawler service is not initialized")
	}

	if len(cfgs) == 0 {
		return fmt.Errorf("no providers configured for crawling")
	}

	errs := s.runAll(ctx, cfgs)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (s *Service) runAll(ctx context.Context, cfgs []providers.Provider) []error {
	workerCount := min(len(cfgs), maxProviderWorkers)
	if workerCount == 0 {
		return nil
	}

	cfgCh := make(chan providers.Provider)
	errCh := make(chan error, len(cfgs))

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, cfgCh, errCh)
		}()
	}

	for _, cfg := range cfgs {
		if ctx.Err() != nil {
			break
		}
		cfgCh <- cfg
	}
	close(cfgCh)

	wg.Wait()
	close(errCh)

	errs := make([]error, 0, len(cfgs))
	for err := range errCh {
		errs = append(errs, err)
	}

	return errs
}

func (s *Service) worker(ctx context.Context, cfgCh <-chan providers.Provider, errCh chan<- error) {
	for cfg := range cfgCh {
		if ctx.Err() != nil {
			return
		}
		if err := s.runProvider(ctx, cfg); err != nil {
			errCh <- err
			logger.ErrorObj("provider crawl failed", "provider_error", map[string]any{
				"provider_id": cfg.ID,
				"error":       err.Error(),
			})
		}
	}
}

func (s *Service) runProvider(ctx context.Context, cfg providers.Provider) error {
	start := time.Now()
	fetcher, err := s.registry.FetcherFor(cfg)
	if err != nil {
		return fmt.Errorf("resolve fetcher for provider %s: %w", cfg.ID, err)
	}

	articles, err := fetcher.Fetch(ctx, cfg)
	if err != nil {
		return fmt.Errorf("fetch provider %s: %w", cfg.ID, err)
	}

	if s.scraper != nil {
		articles = s.scraper.Enrich(ctx, cfg, articles)
	}

	logger.InfoObj("provider crawl completed", "provider_result", map[string]any{
		"provider_id":        cfg.ID,
		"articles_collected": len(articles),
		"elapsed_ms":         time.Since(start).Milliseconds(),
	})
	return nil
}

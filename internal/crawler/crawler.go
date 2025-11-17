package crawler

import (
	"context"
	"errors"
	"fmt"

	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
)

// Service coordinates crawling across multiple providers.
type Service struct {
	registry providers.FetcherRegistry
	scraper  ArticleScraper
}

// NewService wires a crawler with the provider fetcher registry.
func NewService(reg providers.FetcherRegistry) *Service {
	return &Service{
		registry: reg,
		scraper:  NewScraper(nil),
	}
}

// Run executes a crawl pass for all configured providers.
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
	errs := make([]error, 0, len(cfgs))

	for _, cfg := range cfgs {
		if err := s.runProvider(ctx, cfg); err != nil {
			errs = append(errs, err)
			logger.ErrorObj("provider crawl failed", "provider_error", map[string]any{
				"provider_id": cfg.ID,
				"error":       err.Error(),
			})
		}
	}

	return errs
}

func (s *Service) runProvider(ctx context.Context, cfg providers.Provider) error {
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
	})
	return nil
}

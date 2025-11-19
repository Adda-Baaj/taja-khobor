package crawler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
	"github.com/Adda-Baaj/taja-khobor/pkg/publishers"
)

const maxProviderWorkers = 10

type Service struct {
	processor *ProviderProcessor
	log       logger.Logger
}

func NewService(reg providers.FetcherRegistry, pub EventPublisher, log logger.Logger) *Service {
	if log == nil {
		log = logger.NopLogger{}
	}
	processor := NewProviderProcessor(reg, NewScraper(nil, log), pub, log)
	return &Service{
		processor: processor,
		log:       log,
	}
}

func (s *Service) Run(ctx context.Context, cfgs []providers.Provider) error {
	if s == nil || s.processor == nil {
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
		if err := s.processor.Process(ctx, cfg); err != nil {
			errCh <- err
			s.log.ErrorObj("provider crawl failed", "provider_error", map[string]any{
				"provider_id": cfg.ID,
				"error":       err.Error(),
			})
		}
	}
}

// ProviderProcessor fetches, enriches, and publishes provider articles.
type ProviderProcessor struct {
	registry  providers.FetcherRegistry
	scraper   ArticleScraper
	publisher EventPublisher
	log       logger.Logger
}

func NewProviderProcessor(reg providers.FetcherRegistry, scraper ArticleScraper, pub EventPublisher, log logger.Logger) *ProviderProcessor {
	if log == nil {
		log = logger.NopLogger{}
	}
	return &ProviderProcessor{
		registry:  reg,
		scraper:   scraper,
		publisher: pub,
		log:       log,
	}
}

func (p *ProviderProcessor) Process(ctx context.Context, cfg providers.Provider) error {
	if p == nil || p.registry == nil {
		return fmt.Errorf("provider processor not initialized")
	}

	start := time.Now()
	fetcher, err := p.registry.FetcherFor(cfg)
	if err != nil {
		return fmt.Errorf("resolve fetcher for provider %s: %w", cfg.ID, err)
	}

	articles, err := fetcher.Fetch(ctx, cfg)
	if err != nil {
		return fmt.Errorf("fetch provider %s: %w", cfg.ID, err)
	}

	if p.scraper != nil {
		articles = p.scraper.Enrich(ctx, cfg, articles)
	}

	published := 0
	if count, err := p.publishArticles(ctx, cfg, articles); err != nil {
		return fmt.Errorf("publish provider %s articles: %w", cfg.ID, err)
	} else {
		published = count
	}

	p.log.InfoObj("provider crawl completed", "provider_result", map[string]any{
		"provider_id":        cfg.ID,
		"articles_collected": len(articles),
		"articles_published": published,
		"elapsed_ms":         time.Since(start).Milliseconds(),
	})
	return nil
}

func (p *ProviderProcessor) publishArticles(ctx context.Context, cfg providers.Provider, articles []domain.Article) (int, error) {
	if p.publisher == nil || len(articles) == 0 {
		return 0, nil
	}

	var errs []error
	published := 0
	for _, art := range articles {
		evt := publishers.NewEvent(cfg.ID, cfg.Name, art)
		if err := p.publisher.Publish(ctx, evt); err != nil {
			errs = append(errs, fmt.Errorf("article %s: %w", art.ID, err))
			p.log.ErrorObj("failed to publish article", "publisher_error", map[string]any{
				"provider_id": cfg.ID,
				"article_id":  art.ID,
				"error":       err.Error(),
			})
			continue
		}
		published++
	}

	return published, errors.Join(errs...)
}

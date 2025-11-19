package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/config"
	"github.com/Adda-Baaj/taja-khobor/internal/crawler"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
	"github.com/Adda-Baaj/taja-khobor/pkg/publishers"
)

// Collector wires together providers, crawler, and publishers and executes crawl loops.
type Collector struct {
	cfg           *config.Config
	providerReg   *providers.Registry
	fanout        *publishers.Fanout
	crawlService  *crawler.Service
	crawlInterval time.Duration
	log           logger.Logger
}

// NewCollector builds a collector runtime from config files.
func NewCollector(cfg *config.Config, log logger.Logger) (*Collector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if log == nil {
		log = &logger.NopLogger{}
	}

	providerReg, err := providers.LoadRegistry(cfg.ProvidersFile)
	if err != nil {
		return nil, fmt.Errorf("load providers registry: %w", err)
	}
	log.InfoObj("providers registry loaded", "providers", providerReg.All())

	publisherReg, err := publishers.LoadRegistry(cfg.PublishersFile)
	if err != nil {
		return nil, fmt.Errorf("load publishers registry: %w", err)
	}

	enabledPublishers := publisherReg.Enabled()
	if len(enabledPublishers) == 0 {
		return nil, fmt.Errorf("no publishers configured")
	}

	pubRegistry := publishers.DefaultRegistry()
	pubClients, err := publishers.BuildAll(pubRegistry, enabledPublishers)
	if err != nil {
		return nil, fmt.Errorf("build publishers: %w", err)
	}
	fanout := publishers.NewFanout(pubClients)
	log.InfoObj("publishers registry loaded", "publishers", enabledPublishers)

	return &Collector{
		cfg:           cfg,
		providerReg:   providerReg,
		fanout:        fanout,
		crawlService:  crawler.NewService(providers.DefaultFetcherRegistry(nil), fanout, log),
		crawlInterval: cfg.CrawlInterval,
		log:           log,
	}, nil
}

// Run starts the crawl loop until the context is cancelled.
func (c *Collector) Run(ctx context.Context) error {
	if c == nil || c.crawlService == nil {
		return fmt.Errorf("collector is not initialized")
	}

	providers := c.providerReg.All()
	if len(providers) == 0 {
		c.log.WarnObj("no providers configured; collector idle", "providers_file", c.cfg.ProvidersFile)
		<-ctx.Done()
		return ctx.Err()
	}

	c.log.InfoObj("collector loop starting", "collector_state", map[string]any{
		"providers_count":  len(providers),
		"publishers_count": c.fanout.Size(),
		"crawl_interval":   c.crawlInterval.String(),
	})

	if err := c.runOnce(ctx, providers); err != nil {
		return fmt.Errorf("initial crawl: %w", err)
	}

	ticker := time.NewTicker(c.crawlInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.InfoObj("collector loop exiting", "reason", ctx.Err())
			return nil
		case <-ticker.C:
			if err := c.runOnce(ctx, providers); err != nil {
				c.log.ErrorObj("scheduled crawl failed", "error", err)
			}
		}
	}
}

func (c *Collector) runOnce(ctx context.Context, providers []providers.Provider) error {
	start := time.Now()
	c.log.InfoObj("crawl started", "crawl_meta", map[string]any{
		"providers_count": len(providers),
		"started_at":      start.UTC(),
	})
	if err := c.crawlService.Run(ctx, providers); err != nil {
		return err
	}
	c.log.InfoObj("crawl completed", "crawl_meta", map[string]any{
		"providers_count": len(providers),
		"elapsed_ms":      time.Since(start).Milliseconds(),
	})
	return nil
}

package crawler

import (
	"context"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
	"github.com/Adda-Baaj/taja-khobor/pkg/publishers"
)

// ArticleScraper enriches crawled articles with metadata (e.g., OG tags).
type ArticleScraper interface {
	Enrich(ctx context.Context, cfg providers.Provider, articles []domain.Article) []domain.Article
}

// EventPublisher publishes enriched articles downstream.
type EventPublisher interface {
	Publish(ctx context.Context, evt publishers.Event) error
}

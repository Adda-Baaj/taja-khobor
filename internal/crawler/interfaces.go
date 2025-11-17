package crawler

import (
	"context"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"
)

// ArticleScraper enriches crawled articles with metadata (e.g., OG tags).
type ArticleScraper interface {
	Enrich(ctx context.Context, cfg providers.Provider, articles []domain.Article) []domain.Article
}

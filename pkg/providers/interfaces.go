package providers

import (
	"context"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
)

// Fetcher is responsible for retrieving and extracting articles for a provider.
// Concrete implementations live in provider-specific files (e.g., ndtv.go).
type Fetcher interface {
	ID() string
	Fetch(ctx context.Context, cfg Provider) ([]domain.Article, error)
}

// FetcherRegistry resolves the fetcher implementation for a given provider config.
type FetcherRegistry interface {
	FetcherFor(cfg Provider) (Fetcher, error)
}

// HTTPClient aliases the shared httpclient.Client interface for clarity within providers.
type HTTPClient = httpclient.Client

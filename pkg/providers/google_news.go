package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
)

// googleNewsFetcher implements Fetcher for Google News sitemap providers.
type googleNewsFetcher struct {
	client HTTPClient
}

// NewGoogleNewsFetcher builds a Fetcher for Google News sitemap providers.
func NewGoogleNewsFetcher(client HTTPClient) Fetcher {
	if client == nil {
		client = DefaultHTTPClient()
	}
	return &googleNewsFetcher{client: client}
}

// ID returns the provider type for the Google News fetcher.
func (f *googleNewsFetcher) ID() string {
	return ProviderTypeGoogleNews
}

// Fetch retrieves articles from a Google News sitemap provider.
func (f *googleNewsFetcher) Fetch(ctx context.Context, cfg Provider) ([]domain.Article, error) {
	if !strings.EqualFold(cfg.Type, ProviderTypeGoogleNews) {
		return nil, fmt.Errorf("google news fetcher received incompatible provider type %q", cfg.Type)
	}
	if strings.TrimSpace(cfg.SourceURL) == "" {
		return nil, fmt.Errorf("provider %q source_url is empty", cfg.ID)
	}

	headers := Headers(cfg)

	raw, err := fetchSitemap(ctx, f.client, cfg.SourceURL, cfg.ID, headers)
	if err != nil {
		return nil, err
	}

	urls, err := parseGoogleNewsSitemap(raw)
	if err != nil {
		return nil, fmt.Errorf("decode google news sitemap: %w", err)
	}
	articles := buildArticlesFromSitemap(cfg.ID, urls)
	if len(articles) == 0 {
		return nil, fmt.Errorf("%s sitemap returned no records", cfg.ID)
	}
	return articles, nil
}

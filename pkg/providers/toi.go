package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
)

const toiProviderID = "toi"

// toiFetcher fetches Google News sitemap entries for Times of India.
type toiFetcher struct {
	client HTTPClient
}

// NewTOIFetcher builds a fetcher for Times of India sitemap entries.
func NewTOIFetcher(client HTTPClient) Fetcher {
	if client == nil {
		client = DefaultHTTPClient()
	}
	return &toiFetcher{client: client}
}

func (f *toiFetcher) ID() string {
	return toiProviderID
}

func (f *toiFetcher) Fetch(ctx context.Context, cfg Provider) ([]domain.Article, error) {
	if !strings.EqualFold(cfg.ID, toiProviderID) {
		return nil, fmt.Errorf("toi fetcher received incompatible provider %q", cfg.ID)
	}
	if strings.TrimSpace(cfg.SourceURL) == "" {
		return nil, fmt.Errorf("toi provider source_url is empty")
	}

	raw, err := f.download(ctx, cfg)
	if err != nil {
		return nil, err
	}

	urls, err := parseGoogleNewsSitemap(raw)
	if err != nil {
		return nil, err
	}

	articles := make([]domain.Article, 0, len(urls))
	for _, entry := range urls {
		loc := strings.TrimSpace(entry.Loc)
		if loc == "" {
			continue
		}
		title := strings.TrimSpace(entry.NewsTitle)
		if title == "" {
			title = loc
		}

		articles = append(articles, domain.Article{
			ID:    hashURL(loc),
			Title: title,
			URL:   loc,
		})
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("toi sitemap returned no records")
	}

	return articles, nil
}

func (f *toiFetcher) download(ctx context.Context, cfg Provider) ([]byte, error) {
	headers := Headers(cfg)

	resp, err := f.client.Get(ctx, cfg.SourceURL, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch toi sitemap: %w", err)
	}

	body := resp.Body()
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("toi sitemap returned status %d body: %s", resp.StatusCode(), responseSnippet(body))
	}

	return body, nil
}

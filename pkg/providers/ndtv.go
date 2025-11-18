package providers

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
)

const (
	ndtvProviderID = "ndtv"
)

// ndtvFetcher fetches Google News sitemap entries for NDTV.
type ndtvFetcher struct {
	client HTTPClient
	now    func() time.Time
}

// NewNDTVFetcher builds a fetcher for NDTV sitemap entries.
func NewNDTVFetcher(client HTTPClient) Fetcher {
	if client == nil {
		client = DefaultHTTPClient()
	}
	return &ndtvFetcher{
		client: client,
		now:    time.Now,
	}
}

func (f *ndtvFetcher) ID() string {
	return ndtvProviderID
}

func (f *ndtvFetcher) Fetch(ctx context.Context, cfg Provider) ([]domain.Article, error) {
	if !strings.EqualFold(cfg.ID, ndtvProviderID) {
		return nil, fmt.Errorf("ndtv fetcher received incompatible provider %q", cfg.ID)
	}
	if strings.TrimSpace(cfg.SourceURL) == "" {
		return nil, fmt.Errorf("ndtv provider source_url is empty")
	}

	sourceURL, err := datedNDTVSourceURL(cfg.SourceURL, f.now)
	if err != nil {
		return nil, err
	}

	headers := Headers(cfg)

	raw, err := fetchSitemap(ctx, f.client, sourceURL, ndtvProviderID, headers)
	if err != nil {
		return nil, err
	}

	urls, err := parseGoogleNewsSitemap(raw)
	if err != nil {
		return nil, fmt.Errorf("decode google news sitemap: %w", err)
	}
	articles := buildArticlesFromSitemap(urls)
	if len(articles) == 0 {
		return nil, fmt.Errorf("ndtv sitemap returned no records")
	}
	return articles, nil
}

func datedNDTVSourceURL(raw string, now func() time.Time) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("ndtv provider source_url is empty")
	}

	if now == nil {
		now = time.Now
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse ndtv source_url: %w", err)
	}

	t := now().In(time.FixedZone("IST", 5*60*60+30*60))
	y, m, d := t.Date()

	q := parsed.Query()
	q.Set("yyyy", fmt.Sprintf("%04d", y))
	q.Set("mm", fmt.Sprintf("%02d", int(m)))
	q.Set("dd", fmt.Sprintf("%02d", d))
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
}

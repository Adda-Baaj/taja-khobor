package providers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
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

	raw, err := f.download(ctx, cfg, sourceURL)
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
		return nil, fmt.Errorf("ndtv sitemap returned no records")
	}

	return articles, nil
}

func (f *ndtvFetcher) download(ctx context.Context, cfg Provider, sourceURL string) ([]byte, error) {
	headers := Headers(cfg)

	resp, err := f.client.Get(ctx, sourceURL, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch ndtv sitemap: %w", err)
	}

	body := resp.Body()
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("ndtv sitemap returned status %d body: %s", resp.StatusCode(), responseSnippet(body))
	}

	return body, nil
}

type googleNewsSitemap struct {
	URLs []googleNewsURL `xml:"url"`
}

type googleNewsURL struct {
	Loc       string `xml:"loc"`
	NewsTitle string `xml:"news>title"`
}

func parseGoogleNewsSitemap(data []byte) ([]googleNewsURL, error) {
	var sitemap googleNewsSitemap
	if err := xml.Unmarshal(data, &sitemap); err != nil {
		return nil, fmt.Errorf("decode google news sitemap: %w", err)
	}
	return sitemap.URLs, nil
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

// newNDTVFetcherWithClock is a test helper that injects a custom clock.
func newNDTVFetcherWithClock(client HTTPClient, now func() time.Time) Fetcher {
	if client == nil {
		client = DefaultHTTPClient()
	}
	return &ndtvFetcher{
		client: client,
		now:    now,
	}
}

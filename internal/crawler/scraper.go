package crawler

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"

	"github.com/PuerkitoBio/goquery"
)

const (
	maxHTMLBodyBytes = 1 << 20 // 1 MiB
)

// Scraper fetches article pages and extracts metadata from OG tags.
type Scraper struct {
	client httpclient.Client
}

// NewScraper constructs a scraper with the provided HTTP client (or default).
func NewScraper(client httpclient.Client) *Scraper {
	if client == nil {
		client = providers.DefaultHTTPClient()
	}
	return &Scraper{client: client}
}

// Enrich iterates articles, fetching each page (with throttling) and merging OG metadata.
func (s *Scraper) Enrich(ctx context.Context, cfg providers.Provider, articles []domain.Article) []domain.Article {
	delay := cfg.RequestDelay()
	// seed output with originals so we can return what we have on abort
	out := append([]domain.Article(nil), articles...)

	for i, art := range articles {
		select {
		case <-ctx.Done():
			return out[:i]
		default:
		}

		enriched, err := s.fetchAndParse(ctx, cfg, art)
		if err != nil {
			logger.WarnObj("article metadata scrape failed", "metadata_error", map[string]any{
				"provider_id": cfg.ID,
				"url":         art.URL,
				"error":       err.Error(),
			})
			out[i] = art
		} else {
			out[i] = enriched
		}

		if delay > 0 && i < len(articles)-1 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return out[:i+1]
			case <-timer.C:
			}
		}
	}

	return out
}

func (s *Scraper) fetchAndParse(ctx context.Context, cfg providers.Provider, art domain.Article) (domain.Article, error) {
	headers := providers.Headers(cfg)

	resp, err := s.client.Get(ctx, art.URL, headers)
	if err != nil {
		return art, fmt.Errorf("http fetch: %w", err)
	}

	if resp.StatusCode() != 200 {
		snippet := strings.TrimSpace(string(resp.Body()))
		if len(snippet) > 1024 {
			snippet = snippet[:1024]
		}
		return art, fmt.Errorf("status %d body: %s", resp.StatusCode(), snippet)
	}

	body := resp.Body()
	if len(body) > maxHTMLBodyBytes {
		body = body[:maxHTMLBodyBytes]
	}

	meta, err := parseMeta(body)
	if err != nil {
		return art, err
	}
	updated := art
	if meta.Title != "" {
		updated.Title = meta.Title
	}
	if meta.Description != "" {
		updated.Description = meta.Description
	}
	if meta.ImageURL != "" {
		updated.ImageURL = meta.ImageURL
	}

	return updated, nil
}

func parseMeta(body []byte) (pageMeta, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return pageMeta{}, fmt.Errorf("parse html: %w", err)
	}

	pm := pageMeta{}

	extract := func(sel string) string {
		if node := doc.Find(sel).First(); node.Length() > 0 {
			if val, ok := node.Attr("content"); ok {
				return strings.TrimSpace(val)
			}
		}
		return ""
	}

	pm.Title = firstNonEmpty(
		extract(`meta[property="og:title"]`),
		strings.TrimSpace(doc.Find("title").First().Text()),
	)
	pm.Description = firstNonEmpty(
		extract(`meta[property="og:description"]`),
		extract(`meta[name="description"]`),
	)
	pm.ImageURL = extract(`meta[property="og:image"]`)

	return pm, nil
}

type pageMeta struct {
	Title       string
	Description string
	ImageURL    string
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

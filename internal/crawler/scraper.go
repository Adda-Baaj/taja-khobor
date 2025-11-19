package crawler

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/internal/logger"
	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
	"github.com/Adda-Baaj/taja-khobor/pkg/providers"

	"github.com/PuerkitoBio/goquery"
)

const (
	maxHTMLBodyBytes  = 1 << 20 // 1 MiB
	maxArticleWorkers = 10
)

type Scraper struct {
	client httpclient.Client
	log    logger.Logger
}

func NewScraper(client httpclient.Client, log logger.Logger) *Scraper {
	if client == nil {
		client = providers.DefaultHTTPClient()
	}
	if log == nil {
		log = logger.NopLogger{}
	}
	return &Scraper{client: client, log: log}
}

func (s *Scraper) Enrich(ctx context.Context, cfg providers.Provider, articles []domain.Article) []domain.Article {
	delay := cfg.RequestDelay()
	out := make([]domain.Article, len(articles))
	copy(out, articles) // default to originals so partial results are returned on cancel

	if len(articles) == 0 {
		return out
	}

	workerCount := min(len(articles), maxArticleWorkers)

	var limiter <-chan time.Time
	var ticker *time.Ticker
	if delay > 0 {
		ticker = time.NewTicker(delay)
		limiter = ticker.C
		defer ticker.Stop()
	}

	jobCh := make(chan int)
	var wg sync.WaitGroup

	for range workerCount {
		wg.Add(1)
		go s.articleWorker(ctx, cfg, articles, limiter, jobCh, out, &wg)
	}

	for idx := range articles {
		if ctx.Err() != nil {
			break
		}
		jobCh <- idx
	}
	close(jobCh)

	wg.Wait()

	return out
}

func (s *Scraper) articleWorker(
	ctx context.Context,
	cfg providers.Provider,
	articles []domain.Article,
	limiter <-chan time.Time,
	jobCh <-chan int,
	out []domain.Article,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for idx := range jobCh {
		if ctx.Err() != nil {
			return
		}

		if limiter != nil {
			select {
			case <-ctx.Done():
				return
			case <-limiter:
			}
		}

		art := articles[idx]
		if enriched, err := s.fetchAndParse(ctx, cfg, art); err != nil {
			s.log.WarnObj("article metadata scrape failed", "metadata_error", map[string]any{
				"provider_id": cfg.ID,
				"url":         art.URL,
				"error":       err.Error(),
			})
			out[idx] = art
		} else {
			out[idx] = enriched
		}
	}
}

func (s *Scraper) fetchAndParse(ctx context.Context, cfg providers.Provider, art domain.Article) (domain.Article, error) {
	headers := providers.Headers(cfg)

	s.log.InfoObj("scraping article metadata", "scrape_start", map[string]any{
		"provider_id": cfg.ID,
		"url":         art.URL,
	})

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
		s.log.InfoObj("html body truncated", "truncation", map[string]any{
			"provider_id": cfg.ID,
			"url":         art.URL,
			"original":    len(body),
			"kept":        maxHTMLBodyBytes,
		})
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
		updated.ImageURL = resolveURL(meta.ImageURL, art.URL)
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

func resolveURL(raw, base string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.IsAbs() {
		return parsed.String()
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return raw
	}

	return baseURL.ResolveReference(parsed).String()
}

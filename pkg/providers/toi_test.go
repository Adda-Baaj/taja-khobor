package providers

import (
	"context"
	"testing"
)

func TestTOIFetcherFetchSuccess(t *testing.T) {
	const toiSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url>
    <loc>https://timesofindia.indiatimes.com/article-1</loc>
    <news:news>
      <news:title>TOI Headline 1</news:title>
    </news:news>
  </url>
  <url>
    <loc>https://timesofindia.indiatimes.com/article-2</loc>
  </url>
</urlset>`

	client := mockHTTPClient{
		t: t,
		expect: map[string]string{
			"User-Agent":      "UA",
			"Accept":          "A",
			"Accept-Language": "L",
			"Cache-Control":   "C",
		},
		body: toiSitemap,
	}

	fetcher := NewTOIFetcher(client)
	articles, err := fetcher.Fetch(context.Background(), Provider{
		ID:        toiProviderID,
		SourceURL: "https://timesofindia.indiatimes.com/sitemap/today",
		Config: map[string]any{
			ConfigUserAgentKey:      "UA",
			ConfigAcceptKey:         "A",
			ConfigAcceptLanguageKey: "L",
			ConfigCacheControlKey:   "C",
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}
	if articles[0].Title != "TOI Headline 1" {
		t.Errorf("expected first title to be TOI Headline 1, got %s", articles[0].Title)
	}
	if articles[1].Title != "https://timesofindia.indiatimes.com/article-2" {
		t.Errorf("expected fallback title to be url, got %s", articles[1].Title)
	}
}

func TestTOIFetcherRejectsUnknownProvider(t *testing.T) {
	fetcher := NewTOIFetcher(nil)
	_, err := fetcher.Fetch(context.Background(), Provider{
		ID:        "other",
		SourceURL: "https://example.com",
	})
	if err == nil {
		t.Fatal("expected error for mismatched provider id")
	}
}

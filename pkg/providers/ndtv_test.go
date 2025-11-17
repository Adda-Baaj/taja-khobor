package providers

import (
	"context"
	"testing"
	"time"

	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
)

const sampleSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url>
    <loc>https://www.ndtv.com/article-1</loc>
    <news:news>
      <news:title>Headline 1</news:title>
    </news:news>
  </url>
  <url>
    <loc>https://www.ndtv.com/article-2</loc>
  </url>
</urlset>`

type mockHTTPClient struct {
	t         *testing.T
	expect    map[string]string
	expectURL string
	status    int
	body      string
}

type mockResponse struct {
	body       []byte
	statusCode int
}

func (r mockResponse) Body() []byte    { return r.body }
func (r mockResponse) StatusCode() int { return r.statusCode }

func (m mockHTTPClient) Get(ctx context.Context, url string, headers map[string]string) (httpclient.Response, error) {
	if m.expectURL != "" && url != m.expectURL {
		m.t.Fatalf("expected url %q, got %q", m.expectURL, url)
	}
	for key, want := range m.expect {
		if got := headers[key]; got != want {
			m.t.Fatalf("expected header %s=%q, got %q", key, want, got)
		}
	}
	status := m.status
	if status == 0 {
		status = 200
	}
	return mockResponse{body: []byte(m.body), statusCode: status}, nil
}

func TestNDTVFetcherFetchSuccess(t *testing.T) {
	client := mockHTTPClient{
		t:         t,
		expectURL: "https://www.ndtv.com/sitemap.xml?dd=17&mm=11&yyyy=2025",
		expect: map[string]string{
			"User-Agent":      "UA",
			"Accept":          "A",
			"Accept-Language": "L",
			"Cache-Control":   "C",
		},
		body: sampleSitemap,
	}

	fetcher := newNDTVFetcherWithClock(client, func() time.Time {
		return time.Date(2025, time.November, 17, 10, 0, 0, 0, time.FixedZone("IST", 19800))
	})
	articles, err := fetcher.Fetch(context.Background(), Provider{
		ID:        ndtvProviderID,
		SourceURL: "https://www.ndtv.com/sitemap.xml",
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
	if articles[0].Title != "Headline 1" {
		t.Errorf("expected first title to be Headline 1, got %s", articles[0].Title)
	}
	if articles[1].Title != "https://www.ndtv.com/article-2" {
		t.Errorf("expected fallback title to be url, got %s", articles[1].Title)
	}
}

func TestNDTVFetcherRejectsUnknownProvider(t *testing.T) {
	fetcher := newNDTVFetcherWithClock(nil, func() time.Time { return time.Now() })
	_, err := fetcher.Fetch(context.Background(), Provider{
		ID:        "other",
		SourceURL: "https://example.com",
	})
	if err == nil {
		t.Fatal("expected error for mismatched provider id")
	}
}

func TestNDTVFetcherUsesCustomUserAgent(t *testing.T) {
	const (
		customUA     = "CustomAgent/1.0"
		customAccept = "application/xml"
		customLang   = "en-IN,en;q=0.8"
		customCache  = "max-age=0"
	)
	client := mockHTTPClient{
		t:         t,
		expectURL: "https://www.ndtv.com/sitemap.xml?dd=17&mm=11&yyyy=2025",
		expect: map[string]string{
			"User-Agent":      customUA,
			"Accept":          customAccept,
			"Accept-Language": customLang,
			"Cache-Control":   customCache,
		},
		body: sampleSitemap,
	}

	fetcher := newNDTVFetcherWithClock(client, func() time.Time {
		return time.Date(2025, time.November, 17, 10, 0, 0, 0, time.FixedZone("IST", 19800))
	})
	_, err := fetcher.Fetch(context.Background(), Provider{
		ID:        ndtvProviderID,
		SourceURL: "https://www.ndtv.com/sitemap.xml",
		Config: map[string]any{
			ConfigUserAgentKey:      customUA,
			ConfigAcceptKey:         customAccept,
			ConfigAcceptLanguageKey: customLang,
			ConfigCacheControlKey:   customCache,
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
}

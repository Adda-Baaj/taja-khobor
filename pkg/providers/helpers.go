package providers

import (
	"context"
	"crypto/sha1" //nolint:gosec // non-cryptographic id generation
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/Adda-Baaj/taja-khobor/internal/domain"
	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
)

func hashURL(u string) string {
	sum := sha1.Sum([]byte(u))
	return hex.EncodeToString(sum[:])
}

func responseSnippet(body []byte) string {
	const maxLen = 512
	s := strings.TrimSpace(string(body))
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	if s == "" {
		return "<empty>"
	}
	return s
}

type googleNewsSitemap struct {
	URLs []googleNewsURL `xml:"url"`
}

type googleNewsURL struct {
	Loc string `xml:"loc"`
}

func parseGoogleNewsSitemap(data []byte) ([]googleNewsURL, error) {
	var sitemap googleNewsSitemap
	if err := xml.Unmarshal(data, &sitemap); err != nil {
		return nil, err
	}
	return sitemap.URLs, nil
}

func buildArticlesFromSitemap(urls []googleNewsURL) []domain.Article {
	articles := make([]domain.Article, 0, len(urls))
	for _, entry := range urls {
		loc := strings.TrimSpace(entry.Loc)
		if loc == "" {
			continue
		}

		articles = append(articles, domain.Article{
			ID:    hashURL(loc),
			Title: "", // Title is not taken from sitemap
			URL:   loc,
		})
	}
	return articles
}

func fetchSitemap(ctx context.Context, client httpclient.Client, url, providerID string, headers map[string]string) ([]byte, error) {
	resp, err := client.Get(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch %s sitemap: %w", providerID, err)
	}

	body := resp.Body()
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%s sitemap returned status %d body: %s", providerID, resp.StatusCode(), responseSnippet(body))
	}

	return body, nil
}

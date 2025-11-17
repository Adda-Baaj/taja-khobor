package providers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadProvidersYAML(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "providers.yaml")
	content := `
providers:
  - id: ndtv
    name: NDTV News
    type: https
    source_url: https://www.ndtv.com/sitemap/google-news-sitemap
    response_format: xml
    request_delay_ms: 750
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write providers file: %v", err)
	}

	if err := LoadProviders(file); err != nil {
		t.Fatalf("LoadProviders returned error: %v", err)
	}

	providers := Providers()
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}

	p, ok := ProviderByID("ndtv")
	if !ok {
		t.Fatalf("expected provider id ndtv to be loaded")
	}
	if p.SourceURL != "https://www.ndtv.com/sitemap/google-news-sitemap" {
		t.Fatalf("unexpected source_url: %s", p.SourceURL)
	}
	if p.ResponseFormat != "xml" {
		t.Fatalf("unexpected response_format: %s", p.ResponseFormat)
	}
	if p.RequestDelay() != 750*time.Millisecond {
		t.Fatalf("unexpected request delay: %v", p.RequestDelay())
	}
}

func TestLoadProvidersDuplicateID(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "providers.yaml")
	content := `
providers:
  - id: duplicate
    name: Provider One
    type: https
    source_url: https://p1.example
    response_format: xml
  - id: duplicate
    name: Provider Two
    type: https
    source_url: https://p2.example
    response_format: xml
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write providers file: %v", err)
	}

	if err := LoadProviders(file); err == nil {
		t.Fatalf("expected duplicate provider error, got nil")
	}
}

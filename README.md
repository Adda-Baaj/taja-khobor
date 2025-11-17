# taja-khobor

Taja Khobor is an open-source Go microservice that periodically fetches links from news sitemaps/RSS feeds, extracts basic metadata, and emits lightweight `link_collected` events to a downstream queue. It is designed to be provider-pluggable and easy to run locally or in a container.

## Features
- Provider-agnostic crawling via small fetcher interfaces (each provider in its own file).
- Configurable headers (User-Agent, Accept, etc.) per provider via YAML (no hardcoded defaults).
- Per-provider throttling (`request_delay_ms`) to avoid hammering sources.
- HTML metadata enrichment using goquery (OG/title/description/image) after fetching article links.
- Shared HTTP client abstraction (resty under the hood) for providers and scraper.
- Zap-based structured logging; publisher/storage layers are stubbed for now.

## Quickstart
Prereqs: Go 1.22+.

```bash
cp configs/providers.yaml configs/providers.local.yaml  # optional copy to tweak locally
go run ./cmd/collector
```

Environment defaults (overridable via env vars):
- `PROVIDERS_FILE` (default `./configs/providers.yaml`)
- `PUBLISHERS_FILE` (default `./configs/publishers.yaml`)
- `LOG_LEVEL` (default `info`)
- `CRAWL_INTERVAL` (default `15m`)

## Configuring providers
Providers live in `configs/providers.yaml`. Example (NDTV):

```yaml
providers:
  - id: ndtv
    name: NDTV News
    type: https
    source_url: https://www.ndtv.com/sitemap/google-news-sitemap
    response_format: xml
    request_delay_ms: 500                # throttle between article fetches
    config:
      user_agent: <required>              # you must set this; headers are never defaulted
      accept: <optional>
      accept_language: <optional>
      cache_control: <optional>
```

Add new providers by creating a fetcher that implements `pkg/providers.Fetcher`, then register it in `DefaultFetcherRegistry` and add an entry to the YAML.

## Project layout
```
cmd/collector/          # entrypoint
internal/config         # viper/env config loader
internal/crawler        # orchestrates provider fetchers
internal/logger         # zap setup and helpers
pkg/httpclient          # shared HTTP client interfaces + resty adapter
pkg/providers           # provider registry, fetcher interfaces, provider impls
pkg/publisher           # outbound publisher stub
configs/                # provider and app config examples
```

## Development
- Run tests: `go test ./...` (use `GOCACHE=$(pwd)/.gocache` if your env restricts the default cache path).
- Code style: `gofmt` before submitting.
- Contributions: please open issues/PRs with a clear description and tests where applicable. Keep provider-specific logic in its own file and wire it via the fetcher registry to preserve the pluggable design.

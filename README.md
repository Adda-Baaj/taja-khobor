# taja-khobor

Taja Khobor is an open-source Go microservice that periodically collects news links, enriches them with basic metadata, and emits lightweight events downstream. It is built to stay small, composable, and friendly to new contributors.

## What it does (current state)
- Pluggable providers: each source implements the `pkg/providers.Fetcher` interface and is wired through a registry. Currently supported Google News sitemaps:
  - NDTV
  - Times of India
  - The Hindu
  - Financial Express
  - Anandabazar Patrika
  - Ei Samay
  - Aaj Tak
  - Dainik Jagran
  - Dinamalar
  - Daily Thanthi
- Config-driven crawling: providers are declared in YAML/JSON with per-provider headers (User-Agent, Accept, etc.) and `request_delay_ms` throttling.
- Link extraction: provider fetchers pull URLs from sitemaps/RSS. Common helpers handle Google News sitemap parsing and article ID generation.
- Metadata enrichment: fetched links are optionally enriched from OG/title/description/image tags with goquery; cancellation returns whatever was processed so far.
- Shared HTTP client abstraction (resty under the hood) and centralized header builder.
- Pluggable publishers: registry-driven fan-out to HTTP webhooks or AWS SQS queues with JSON payloads.
- Structured logging with zap; storage layer remains stubbed for contributors to extend.

## Quickstart
Prereqs: Go 1.22+.

```bash
cp configs/providers.yaml configs/providers.local.yaml  # tweak locally if needed
go run ./cmd/collector
```

The collector process stays alive and triggers a new crawl every `CRAWL_INTERVAL` (15 minutes by default).

Environment defaults (overridable via env vars):
- `PROVIDERS_FILE` (default `./configs/providers.yaml`)
- `PUBLISHERS_FILE` (default `./configs/publishers.yaml`)
- `LOG_LEVEL` (default `info`)
- `CRAWL_INTERVAL` (default `15m`)

## Configuring providers
Providers live in `configs/providers.yaml` (YAML or JSON is accepted). Example:

```yaml
providers:
  - id: ndtv
    name: NDTV News
    type: google_news_sitemap
    source_url: https://www.ndtv.com/sitemap/google-news-sitemap
    response_format: xml
    request_delay_ms: 500
    config:
      user_agent: <required>   # always set this; headers are never defaulted
      accept: <optional>
      accept_language: <optional>
      cache_control: <optional>
```

Adding a provider:
1. For another Google News sitemap, just add an entry like above (set `type: google_news_sitemap`).
2. For non-sitemap sources, implement `pkg/providers.Fetcher` in a new file, register it in `pkg/providers.DefaultFetcherRegistry`, and set that provider’s `type` (or override via id) to point at your custom fetcher.

## Project layout
```
cmd/collector/          # entrypoint
internal/app/           # collector runtime wiring config, crawler, and plugins
internal/config         # viper/env config loader
internal/crawler        # orchestrates provider fetchers + enrichment
internal/logger         # zap setup and helpers
pkg/httpclient          # shared HTTP client interfaces + resty adapter
pkg/providers           # provider registry, fetcher interfaces, provider impls, sitemap helpers
pkg/publishers          # pluggable sink interfaces + HTTP/SQS implementations
configs/                # provider and app config examples
```

## Contributing
- Open to PRs and issues; keep changes small and focused.
- Prefer one file per provider, registered via the fetcher registry to preserve pluggability.
- Run `gofmt` and `go test ./...` before submitting.
- Discussions and improvements around storage/backoff are welcome—those layers are intentionally minimal today.

## Configuring publishers
Publishers live in `configs/publishers.yaml` (same file may also be JSON). Each entry declares a sink and can be toggled individually.

```yaml
publishers:
  - id: primary-sqs
    type: sqs
    enabled: true
    sqs:
      uri: https://sqs.ap-south-1.amazonaws.com/1234567890/link-events
      region: ap-south-1

  - id: webhook
    type: http
    enabled: true
    http:
      url: https://example.com/hooks/link-events
      method: POST          # optional, defaults to POST
      timeout_seconds: 5    # optional, defaults to 5 seconds
      headers:
        X-Api-Key: secret
```

Currently supported publisher types:
- `http`: Sends JSON payloads via resty client. Custom headers/method/timeouts supported.
- `sqs`: Sends JSON payloads to AWS SQS with basic message attributes (requires valid AWS creds/resolved env).

Disable a sink by setting `enabled: false`. Unknown/disabled types are ignored so future sinks can be pre-declared.

package publishers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Adda-Baaj/taja-khobor/pkg/httpclient"
	"github.com/go-resty/resty/v2"
)

type httpPublisher struct {
	id      string
	method  string
	url     string
	headers map[string]string
	client  *resty.Client
	typ     string
}

func newHTTPPublisher(cfg PublisherConfig) (Publisher, error) {
	if cfg.HTTP == nil {
		return nil, fmt.Errorf("publisher %q missing http configuration", cfg.ID)
	}

	client := httpclient.NewRestyHTTPClient(time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second)

	return &httpPublisher{
		id:      cfg.ID,
		typ:     TypeHTTP,
		method:  cfg.HTTP.Method,
		url:     cfg.HTTP.URL,
		headers: cfg.HTTP.Headers,
		client:  client,
	}, nil
}

func (h *httpPublisher) ID() string   { return h.id }
func (h *httpPublisher) Type() string { return h.typ }

func (h *httpPublisher) Publish(ctx context.Context, evt Event) error {
	req := h.client.R().
		SetContext(ctx).
		SetBody(evt)

	if len(h.headers) > 0 {
		req.SetHeaders(h.headers)
	}

	req.SetHeader("Content-Type", "application/json")

	resp, err := req.Execute(h.method, h.url)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	if resp.IsError() {
		snippet := readBodySnippet(resp.Body())
		return fmt.Errorf("http response status %d: %s", resp.StatusCode(), snippet)
	}
	return nil
}

func readBodySnippet(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if len(body) > 512 {
		body = body[:512]
	}
	return strings.TrimSpace(string(body))
}

package httpclient

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
)

// RestyClient adapts resty.Client to the httpclient.Client interface.
type RestyClient struct {
	client *resty.Client
}

// NewRestyClient creates a new RestyClient with the specified timeout.
func NewRestyClient(timeout time.Duration) *RestyClient {
	return &RestyClient{client: newRestyBaseClient(timeout)}
}

// NewRestyHTTPClient exposes a configured resty.Client for callers needing custom verbs.
func NewRestyHTTPClient(timeout time.Duration) *resty.Client {
	return newRestyBaseClient(timeout)
}

// newRestyBaseClient creates a new resty.Client with the specified timeout.
func newRestyBaseClient(timeout time.Duration) *resty.Client {
	c := resty.New()
	c.SetTimeout(timeout)
	return c
}

// Get performs an HTTP GET request with the specified context, URL, and headers.
func (r *RestyClient) Get(ctx context.Context, url string, headers map[string]string) (Response, error) {
	req := r.client.R().SetContext(ctx)
	if len(headers) > 0 {
		req.SetHeaders(headers)
	}
	resp, err := req.Get(url)
	if err != nil {
		return nil, err
	}
	return &restyResponseAdapter{resp: resp}, nil
}

// restyResponseAdapter adapts resty.Response to the httpclient.Response interface.
type restyResponseAdapter struct {
	resp *resty.Response
}

func (r *restyResponseAdapter) Body() []byte    { return r.resp.Body() }
func (r *restyResponseAdapter) StatusCode() int { return r.resp.StatusCode() }

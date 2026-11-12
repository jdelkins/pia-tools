package httpclient

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

const Timeout = 30 * time.Second

// New returns an isolated, pooled client with a timeout that includes response
// body reads. Callers should reuse the client across requests.
func New() *http.Client {
	client := cleanhttp.DefaultPooledClient()
	client.Timeout = Timeout
	return client
}

// WithTimeout bounds requests for libraries that accept a transport but not an
// HTTP client. The deadline remains active until the response body is closed.
func WithTimeout(base http.RoundTripper, timeout time.Duration) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &timeoutTransport{base: base, timeout: timeout}
}

type timeoutTransport struct {
	base    http.RoundTripper
	timeout time.Duration
}

func (t *timeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
	resp, err := t.base.RoundTrip(req.WithContext(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (body *cancelBody) Close() error {
	body.cancel()
	return body.ReadCloser.Close()
}

package httpclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := New()
	defer client.CloseIdleConnections()
	if client.Timeout != Timeout || client.Timeout <= 0 {
		t.Errorf("client timeout = %s, want %s", client.Timeout, Timeout)
	}
	if client.Transport == nil || client.Transport == http.DefaultTransport {
		t.Error("expected an isolated transport")
	}
}

func TestTimeoutIncludesResponseBody(t *testing.T) {
	for _, mode := range []string{"client", "transport"} {
		for _, stage := range []string{"headers", "body"} {
			t.Run(mode+"/"+stage, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if stage == "body" {
						w.Header().Set("Content-Length", "100")
						w.WriteHeader(http.StatusOK)
						w.(http.Flusher).Flush()
					}
					<-r.Context().Done()
				}))
				defer server.Close()

				client := New()
				defer client.CloseIdleConnections()
				if mode == "client" {
					client.Timeout = 50 * time.Millisecond
				} else {
					client.Timeout = 0
					client.Transport = WithTimeout(client.Transport, 50*time.Millisecond)
				}
				// A longer caller deadline keeps a regression from hanging the test.
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				start := time.Now()
				resp, err := client.Do(req)
				if err == nil {
					_, err = io.ReadAll(resp.Body)
					resp.Body.Close()
				}
				var timeoutError net.Error
				if !errors.As(err, &timeoutError) || !timeoutError.Timeout() {
					t.Fatalf("expected timeout waiting for %s, got %v", stage, err)
				}
				if elapsed := time.Since(start); elapsed >= time.Second {
					t.Errorf("request used caller deadline rather than request timeout: %s", elapsed)
				}
			})
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTransportPreservesEarlierDeadlineAndCancelsOnClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wantDeadline, _ := ctx.Deadline()
	var requestContext context.Context
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestContext = req.Context()
		deadline, ok := requestContext.Deadline()
		if !ok || !deadline.Equal(wantDeadline) {
			t.Errorf("request deadline = %v, want %v", deadline, wantDeadline)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := WithTimeout(base, Timeout).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if requestContext.Err() != nil {
		t.Fatal("response context canceled before reading body")
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(requestContext.Err(), context.Canceled) {
		t.Error("closing response body did not release its context")
	}
	if ctx.Err() != nil {
		t.Error("closing response body canceled caller context")
	}
}

func TestTransportCancelsOnError(t *testing.T) {
	wantErr := errors.New("transport failed")
	var requestContext context.Context
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestContext = req.Context()
		return nil, wantErr
	})
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithTimeout(base, Timeout).RoundTrip(req); !errors.Is(err, wantErr) {
		t.Errorf("RoundTrip error = %v, want %v", err, wantErr)
	}
	if !errors.Is(requestContext.Err(), context.Canceled) {
		t.Error("transport failure did not release request context")
	}
}

package transmission

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jdelkins/pia-tools/internal/httpclient"
)

func TestRPCTimeout(t *testing.T) {
	original := httpClient
	httpClient = httpclient.New()
	httpClient.Timeout = 50 * time.Millisecond
	t.Cleanup(func() {
		httpClient.CloseIdleConnections()
		httpClient = original
	})
	for _, operation := range []string{"notify", "confirm"} {
		for _, stage := range []string{"headers", "body"} {
			t.Run(operation+"/"+stage, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					if stage == "body" {
						w.Header().Set("Content-Length", "100")
						w.WriteHeader(http.StatusOK)
						w.(http.Flusher).Flush()
					}
					select {
					case <-r.Context().Done():
					case <-time.After(2 * time.Second):
					}
				}))
				defer server.Close()
				start := time.Now()
				var err error
				if operation == "notify" {
					err = Notify(server.URL, "", "", 12345)
				} else {
					_, err = Confirm(server.URL, "", "")
				}
				if err == nil {
					t.Fatal("expected RPC timeout")
				}
				if elapsed := time.Since(start); elapsed >= time.Second {
					t.Errorf("RPC timeout took too long: %s", elapsed)
				}
			})
		}
	}
}

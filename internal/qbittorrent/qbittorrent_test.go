package qbittorrent

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jdelkins/pia-tools/internal/httpclient"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		cookies []http.Cookie
		wantErr bool
	}{
		{
			name:    "documented success status",
			status:  http.StatusOK,
			cookies: []http.Cookie{{Name: "SID", Value: "session-id", Path: "/"}},
		},
		{
			name:    "legacy no content status",
			status:  http.StatusNoContent,
			cookies: []http.Cookie{{Name: "SID", Value: "session-id", Path: "/"}},
		},
		{
			name:   "additional cookies",
			status: http.StatusOK,
			cookies: []http.Cookie{
				{Name: "other", Value: "other-value"},
				{Name: "SID", Value: "session-id", Path: "/"},
			},
		},
		{
			name:    "missing session cookie",
			status:  http.StatusOK,
			wantErr: true,
		},
		{
			name:    "unrelated cookie only",
			status:  http.StatusOK,
			cookies: []http.Cookie{{Name: "other", Value: "other-value"}},
			wantErr: true,
		},
		{
			name:    "empty session cookie",
			status:  http.StatusOK,
			cookies: []http.Cookie{{Name: "SID", Value: ""}},
			wantErr: true,
		},
		{
			name:    "forbidden",
			status:  http.StatusForbidden,
			wantErr: true,
		},
		{
			name:    "server error with cookie",
			status:  http.StatusInternalServerError,
			cookies: []http.Cookie{{Name: "SID", Value: "session-id"}},
			wantErr: true,
		},
		{
			name:    "unexpected success status",
			status:  http.StatusCreated,
			cookies: []http.Cookie{{Name: "SID", Value: "session-id"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v2/auth/login" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
					t.Errorf("Content-Type = %q", got)
				}
				if err := r.ParseForm(); err != nil {
					t.Errorf("parse login form: %v", err)
				}
				if r.PostForm.Get("username") != "test user" || r.PostForm.Get("password") != "p&ss=word" {
					t.Error("login credentials were not encoded correctly")
				}
				for _, cookie := range tt.cookies {
					http.SetCookie(w, &cookie)
				}
				w.WriteHeader(tt.status)
				if tt.status == http.StatusOK {
					if tt.wantErr {
						_, _ = w.Write([]byte("Fails."))
					} else {
						_, _ = w.Write([]byte("Ok."))
					}
				}
			}))
			defer server.Close()

			cookie, err := Login(server.URL+"/api/v2", "test user", "p&ss=word")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected login error")
				}
				if cookie != nil {
					t.Errorf("returned cookie on failed login: %v", cookie)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login: %v", err)
			}
			if cookie == nil || cookie.Name != "SID" || cookie.Value != "session-id" || cookie.Path != "/" {
				t.Errorf("unexpected session cookie: %v", cookie)
			}
		})
	}
}

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackedBody struct {
	io.Reader
	read   int
	closed bool
}

func (body *trackedBody) Read(p []byte) (int, error) {
	n, err := body.Reader.Read(p)
	body.read += n
	return n, err
}

func (body *trackedBody) Close() error {
	body.closed = true
	return nil
}

func TestResponsesAreDrainedAndClosed(t *testing.T) {
	operations := []struct {
		name string
		call func() error
	}{
		{"login", func() error {
			_, err := Login("http://example.com/api/v2", "user", "password")
			return err
		}},
		{"logout", func() error {
			return Logout("http://example.com/api/v2", &http.Cookie{Name: "SID", Value: "session-id"})
		}},
		{"notify", func() error {
			return Notify("http://example.com/api/v2", "api-key", nil, 12345)
		}},
		{"confirm", func() error {
			_, err := Confirm("http://example.com/api/v2", "api-key", nil)
			return err
		}},
	}
	for _, operation := range operations {
		for _, success := range []bool{false, true} {
			name := "error"
			status := http.StatusForbidden
			if success {
				name = "success"
				status = http.StatusOK
			}
			t.Run(operation.name+"/"+name, func(t *testing.T) {
				const payload = `{"listen_port":12345}`
				body := &trackedBody{Reader: strings.NewReader(payload)}
				original := httpClient
				httpClient = &http.Client{Transport: responseTransport(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Header:     http.Header{"Set-Cookie": {"SID=session-id; Path=/"}},
						Body:       body,
					}, nil
				})}
				t.Cleanup(func() { httpClient = original })
				if err := operation.call(); (err == nil) != success {
					t.Errorf("unexpected operation result: %v", err)
				}
				if !body.closed {
					t.Error("response body was not closed")
				}
				if body.read != len(payload) {
					t.Errorf("read %d response bytes, want %d", body.read, len(payload))
				}
			})
		}
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) {
	return 0, errors.New("response read failed")
}

func TestConfirmClosesBodyOnDecodeAndReadErrors(t *testing.T) {
	for _, reader := range []io.Reader{strings.NewReader("invalid JSON"), failedReader{}} {
		body := &trackedBody{Reader: reader}
		original := httpClient
		httpClient = &http.Client{Transport: responseTransport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		})}
		_, err := Confirm("http://example.com/api/v2", "api-key", nil)
		httpClient = original
		if err == nil {
			t.Error("expected response parsing error")
		}
		if !body.closed {
			t.Error("response body was not closed after parsing error")
		}
	}
}

func TestResponseDrainIsBounded(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader(strings.Repeat("x", 8192))}
	closeResponseBody(body)
	if !body.closed {
		t.Error("large response body was not closed")
	}
	if body.read != 4096 {
		t.Errorf("drained %d bytes, want 4096", body.read)
	}
}

func TestWorkflowReusesConnection(t *testing.T) {
	addresses := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		addresses <- r.RemoteAddr
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session-id", Path: "/"})
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/app/preferences":
			_, _ = w.Write([]byte(`{"listen_port":12345}`))
		default:
			_, _ = w.Write([]byte("Ok."))
		}
	}))
	defer server.Close()
	original := httpClient
	httpClient = httpclient.New()
	defer func() {
		httpClient.CloseIdleConnections()
		httpClient = original
	}()

	url := server.URL + "/api/v2"
	cookie, err := Login(url, "user", "password")
	if err != nil {
		t.Fatal(err)
	}
	if err := Notify(url, "", cookie, 12345); err != nil {
		t.Fatal(err)
	}
	port, err := Confirm(url, "", cookie)
	if err != nil || port != 12345 {
		t.Fatalf("Confirm: port=%d, error=%v", port, err)
	}
	if err := Logout(url, cookie); err != nil {
		t.Fatal(err)
	}
	first := <-addresses
	for range 3 {
		if got := <-addresses; got != first {
			t.Errorf("connection was not reused: got %q, want %q", got, first)
		}
	}
}

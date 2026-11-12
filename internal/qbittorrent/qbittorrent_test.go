package qbittorrent

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

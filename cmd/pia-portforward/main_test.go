package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/jdelkins/pia-tools/internal/pia"
)

func clearCLIEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"PIA_USERNAME", "PIA_PASSWORD", "RTORRENT", "TRANSMISSION",
		"TRANSMISSION_USERNAME", "TRANSMISSION_PASSWORD", "QBITTORRENT",
		"QBITTORRENT_USERNAME", "QBITTORRENT_PASSWORD", "QBITTORRENT_APIKEY",
	} {
		t.Setenv(name, "")
	}
}

func assertNotifyResult(t *testing.T, cli *CLI, wantPanic string) {
	t.Helper()
	failure := func() (failure any) {
		defer func() { failure = recover() }()
		notifyClients(cli, &pia.Tunnel{PFSig: pia.PortForwardSig{Port: 12345}})
		return
	}()
	if wantPanic == "" {
		if failure != nil {
			t.Fatalf("unexpected failure: %v", failure)
		}
	} else if failure == nil || !strings.Contains(fmt.Sprint(failure), wantPanic) {
		t.Errorf("failure = %v, want %q", failure, wantPanic)
	}
}

func parseCLI(t *testing.T, args ...string) CLI {
	t.Helper()
	var cli CLI
	parser, err := kong.New(&cli, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse(args); err != nil {
		t.Fatal(err)
	}
	return cli
}

func TestCLIDefaults(t *testing.T) {
	clearCLIEnv(t)
	got := parseCLI(t)
	want := CLI{IfName: "pia", CacheDir: "/var/cache/pia"}
	if got != want {
		t.Errorf("CLI defaults = %#v, want %#v", got, want)
	}
}

func TestCLIFlagsAndAliases(t *testing.T) {
	clearCLIEnv(t)
	got := parseCLI(t,
		"--ifname", "wg_test", "--cachedir", "/test/cache", "-r",
		"-u", "pia-user", "-p", "pia-password",
		"--rtorrent", "http://rtorrent", "--transmission", "http://transmission/rpc",
		"--transmission-username", "trans-user", "--transmission-password", "trans-password",
		"--qbittorrent", "http://qbittorrent/api/v2", "--qbittorrent-username", "qbt-user",
		"--qbittorrent-password", "qbt-password", "--qbittorrent-apikey", "qbt-key",
	)
	want := CLI{
		IfName: "wg_test", CacheDir: "/test/cache", Refresh: true,
		Username: "pia-user", Password: "pia-password", Rtorrent: "http://rtorrent",
		Transmission: "http://transmission/rpc", TransUser: "trans-user", TransPassword: "trans-password",
		Qbittorrent: "http://qbittorrent/api/v2", QbtUser: "qbt-user", QbtPassword: "qbt-password", QbtApiKey: "qbt-key",
	}
	if got != want {
		t.Errorf("CLI flags = %#v, want %#v", got, want)
	}
}

func TestCLIEnvironmentAndFlagPrecedence(t *testing.T) {
	clearCLIEnv(t)
	for name, value := range map[string]string{
		"PIA_USERNAME": "pia-user", "PIA_PASSWORD": "pia-password", "RTORRENT": "http://rtorrent",
		"TRANSMISSION": "http://transmission/rpc", "TRANSMISSION_USERNAME": "trans-user", "TRANSMISSION_PASSWORD": "trans-password",
		"QBITTORRENT": "http://qbittorrent/api/v2", "QBITTORRENT_USERNAME": "qbt-user",
		"QBITTORRENT_PASSWORD": "qbt-password", "QBITTORRENT_APIKEY": "qbt-key",
	} {
		t.Setenv(name, value)
	}
	want := CLI{
		IfName: "pia", CacheDir: "/var/cache/pia", Username: "pia-user", Password: "pia-password",
		Rtorrent: "http://rtorrent", Transmission: "http://transmission/rpc", TransUser: "trans-user", TransPassword: "trans-password",
		Qbittorrent: "http://qbittorrent/api/v2", QbtUser: "qbt-user", QbtPassword: "qbt-password", QbtApiKey: "qbt-key",
	}
	if got := parseCLI(t); got != want {
		t.Errorf("CLI environment = %#v, want %#v", got, want)
	}
	want.Username = "flag-user"
	want.QbtApiKey = "flag-key"
	if got := parseCLI(t, "--username", want.Username, "--qbittorrent-apikey", want.QbtApiKey); got != want {
		t.Errorf("flags did not override environment: %#v", got)
	}
}

func TestNotifyClientsWithoutEndpoints(t *testing.T) {
	// No downstream service is required to obtain or maintain a forwarded port.
	notifyClients(&CLI{}, &pia.Tunnel{PFSig: pia.PortForwardSig{Port: 12345}})
}

func TestNotifyQbittorrent(t *testing.T) {
	for _, tt := range []struct {
		name        string
		apiKey      string
		credentials bool
		failPath    string
		wrongPort   bool
		wantPanic   string
		wantPaths   []string
	}{
		{name: "session authentication", credentials: true, wantPaths: []string{"auth/login", "app/setPreferences", "app/preferences", "auth/logout"}},
		{name: "API key takes precedence", apiKey: "api-key", credentials: true, wantPaths: []string{"app/setPreferences", "app/preferences"}},
		{name: "authentication bypass", wantPaths: []string{"app/setPreferences", "app/preferences"}},
		{name: "login failure", credentials: true, failPath: "auth/login", wantPanic: "Could not login", wantPaths: []string{"auth/login"}},
		{name: "notification failure", apiKey: "api-key", failPath: "app/setPreferences", wantPanic: "Could not notify", wantPaths: []string{"app/setPreferences"}},
		{name: "confirmation failure", apiKey: "api-key", failPath: "app/preferences", wantPanic: "Could not verify", wantPaths: []string{"app/setPreferences", "app/preferences"}},
		{name: "mismatched port", apiKey: "api-key", wrongPort: true, wantPanic: "reports port is 54321", wantPaths: []string{"app/setPreferences", "app/preferences"}},
		{name: "logout failure", credentials: true, failPath: "auth/logout", wantPanic: "Failed to logout", wantPaths: []string{"auth/login", "app/setPreferences", "app/preferences", "auth/logout"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(chan string, 8)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/api/v2/")
				paths <- path
				if err := r.ParseForm(); err != nil {
					t.Errorf("parse request: %v", err)
				}
				if path != "auth/login" {
					if tt.apiKey != "" {
						if got := r.Header.Get("Authorization"); got != "Bearer "+tt.apiKey {
							t.Errorf("Authorization = %q", got)
						}
					} else if tt.credentials {
						if cookie, err := r.Cookie("SID"); err != nil || cookie.Value != "session-id" {
							t.Errorf("session cookie = %v, %v", cookie, err)
						}
					}
				}
				if path == tt.failPath {
					http.Error(w, "deliberate failure", http.StatusForbidden)
					return
				}
				switch path {
				case "auth/login":
					if r.PostForm.Get("username") != "qbt-user" || r.PostForm.Get("password") != "qbt-password" {
						t.Error("incorrect login credentials")
					}
					http.SetCookie(w, &http.Cookie{Name: "SID", Value: "session-id", Path: "/"})
					_, _ = io.WriteString(w, "Ok.")
				case "app/setPreferences":
					var prefs struct {
						Port int `json:"listen_port"`
					}
					if err := json.Unmarshal([]byte(r.PostForm.Get("json")), &prefs); err != nil || prefs.Port != 12345 {
						t.Errorf("notified port = %d, error=%v", prefs.Port, err)
					}
				case "app/preferences":
					port := 12345
					if tt.wrongPort {
						port = 54321
					}
					_, _ = fmt.Fprintf(w, `{"listen_port":%d}`, port)
				case "auth/logout":
				default:
					t.Errorf("unexpected API path: %s", path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			cli := &CLI{Qbittorrent: server.URL + "/api/v2", QbtApiKey: tt.apiKey}
			if tt.credentials {
				cli.QbtUser, cli.QbtPassword = "qbt-user", "qbt-password"
			}
			assertNotifyResult(t, cli, tt.wantPanic)
			var gotPaths []string
			for len(paths) > 0 {
				gotPaths = append(gotPaths, <-paths)
			}
			if !reflect.DeepEqual(gotPaths, tt.wantPaths) {
				t.Errorf("API calls = %v, want %v", gotPaths, tt.wantPaths)
			}
		})
	}
}

func TestNotifyTransmission(t *testing.T) {
	for _, tt := range []struct {
		name       string
		failMethod string
		wrongPort  bool
		wantPanic  string
		wantCalls  []string
	}{
		{name: "success", wantCalls: []string{"session-set", "session-get"}},
		{name: "notification failure", failMethod: "session-set", wantPanic: "Could not notify transmission", wantCalls: []string{"session-set"}},
		{name: "confirmation failure", failMethod: "session-get", wantPanic: "Could not verify transmission", wantCalls: []string{"session-set", "session-get"}},
		{name: "mismatched port", wrongPort: true, wantPanic: "transmission reports port is 54321", wantCalls: []string{"session-set", "session-get"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if user, password, ok := r.BasicAuth(); !ok || user != "trans-user" || password != "trans-password" {
					t.Error("missing Transmission authentication")
				}
				var request struct {
					Method    string `json:"method"`
					Tag       int    `json:"tag"`
					Arguments struct {
						Port int `json:"peer-port"`
					} `json:"arguments"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode Transmission request: %v", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				calls <- request.Method
				if request.Method == "session-set" && request.Arguments.Port != 12345 {
					t.Errorf("notified port = %d, want 12345", request.Arguments.Port)
				}
				if request.Method == tt.failMethod {
					http.Error(w, "deliberate failure", http.StatusInternalServerError)
					return
				}
				port := 12345
				if tt.wrongPort {
					port = 54321
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"tag": request.Tag, "result": "success", "arguments": map[string]int{"peer-port": port},
				})
			}))
			defer server.Close()
			assertNotifyResult(t, &CLI{Transmission: server.URL, TransUser: "trans-user", TransPassword: "trans-password"}, tt.wantPanic)
			var gotCalls []string
			for len(calls) > 0 {
				gotCalls = append(gotCalls, <-calls)
			}
			if !reflect.DeepEqual(gotCalls, tt.wantCalls) {
				t.Errorf("RPC calls = %v, want %v", gotCalls, tt.wantCalls)
			}
		})
	}
}

func TestNotifyRtorrent(t *testing.T) {
	allCalls := []string{"network.port_range.set", "network.bind_address.set", "network.port_range"}
	for _, tt := range []struct {
		name       string
		failMethod string
		wrongPort  bool
		wantPanic  string
		wantCalls  []string
	}{
		{name: "success", wantCalls: allCalls},
		{name: "notification failure", failMethod: allCalls[0], wantPanic: "Could not notify rtorrent", wantCalls: allCalls[:1]},
		{name: "rebind failure", failMethod: allCalls[1], wantPanic: "Could not notify rtorrent", wantCalls: allCalls[:2]},
		{name: "confirmation failure", failMethod: allCalls[2], wantPanic: "Could not verify rtorrent", wantCalls: allCalls},
		{name: "mismatched port", wrongPort: true, wantPanic: "rtorrent reports port is 54321", wantCalls: allCalls},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/RPC2" {
					t.Errorf("unexpected RPC URL: %s", r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read RPC request: %v", err)
				}
				var request struct {
					Method string `xml:"methodName"`
				}
				if err := xml.Unmarshal(body, &request); err != nil {
					t.Errorf("decode RPC request: %v", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				calls <- request.Method
				if request.Method == allCalls[0] && !strings.Contains(string(body), "12345-12345") {
					t.Error("missing forwarded port range")
				}
				if request.Method == tt.failMethod {
					http.Error(w, "deliberate failure", http.StatusInternalServerError)
					return
				}
				portRange := "12345-12345"
				if tt.wrongPort {
					portRange = "54321-54321"
				}
				_, _ = fmt.Fprintf(w, `<methodResponse><params><param><value><string>%s</string></value></param></params></methodResponse>`, portRange)
			}))
			defer server.Close()
			assertNotifyResult(t, &CLI{Rtorrent: server.URL}, tt.wantPanic)
			var gotCalls []string
			for len(calls) > 0 {
				gotCalls = append(gotCalls, <-calls)
			}
			if !reflect.DeepEqual(gotCalls, tt.wantCalls) {
				t.Errorf("RPC calls = %v, want %v", gotCalls, tt.wantCalls)
			}
		})
	}
}

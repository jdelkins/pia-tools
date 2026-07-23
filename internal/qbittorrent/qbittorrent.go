package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

func Login(server, username, password string) (*http.Cookie, error) {
	reqUrl, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse server url. %s", server)
	}
	reqUrl = reqUrl.JoinPath("auth/login")

	formValues := url.Values{}
	formValues.Add("username", username)
	formValues.Add("password", password)

	var req *http.Request
	if req, err = http.NewRequest("POST", reqUrl.String(), strings.NewReader(formValues.Encode())); err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return nil, fmt.Errorf("Failed to get response from Qbittorrent: %v", err)
	}

	// WebUI docs say only status 200 will be returned, it actually seems to return 204
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("Got non-204 status code %d from Qbittorrent endpoint", resp.StatusCode)
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, fmt.Errorf("Did not recieve session cookie back from Qbittorrent")
	} else if len(cookies) > 1 {
		return nil, fmt.Errorf("Got %d cookies back from Qbittorrent, more than the 1 expected.", len(cookies))
	}
	sessionCookie := cookies[0]
	return sessionCookie, nil
}

func Logout(server string, sessionCookie *http.Cookie) error {
	if sessionCookie == nil {
		return fmt.Errorf("Got nil session cookie.")
	}

	reqUrl, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("Failed to parse server url. %s", server)
	}
	reqUrl = reqUrl.JoinPath("auth/logout")

	var req *http.Request
	if req, err = http.NewRequest("POST", reqUrl.String(), nil); err != nil {
		return fmt.Errorf("Failed to create request: %v", err)
	}
	req.AddCookie(sessionCookie)
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return fmt.Errorf("Failed to get response from Qbittorrent: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Got non-200 status code %d from Qbittorrent endpoint", resp.StatusCode)
	}
	return nil
}

// the relevant qbittorrent Application preferences
type qbPreferences struct {
	ListenPort int `json:"listen_port"`
}

func Notify(server, apikey string, sessionCookie *http.Cookie, port int) error {
	if apikey == "" && sessionCookie == nil {
		return fmt.Errorf("Got nil api key and session cookie.")
	}

	reqUrl, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("Failed to parse server url. %s", server)
	}
	reqUrl = reqUrl.JoinPath("app/setPreferences")

	prefs := qbPreferences{
		ListenPort: port,
	}
	encodedJSON, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("Failed to marshal JSON for preferences: %v", err)
	}

	// qbittorrent wants the preference values encoded as form values.
	formValues := url.Values{}
	formValues.Add("json", string(encodedJSON))
	var req *http.Request
	if req, err = http.NewRequest("POST", reqUrl.String(), strings.NewReader(formValues.Encode())); err != nil {
		return fmt.Errorf("Failed to create request: %v", err)
	}

	if apikey != "" {
		req.Header.Set("Authorization", "Bearer "+apikey)
	} else {
		req.AddCookie(sessionCookie)
	}
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return fmt.Errorf("Failed to get response from Qbittorrent: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Got non-200 status code %d from Qbittorrent", resp.StatusCode)
	}

	return nil
}

func Confirm(server, apikey string, sessionCookie *http.Cookie) (int, error) {
	if apikey == "" && sessionCookie == nil {
		return 0, fmt.Errorf("Got nil api key and session cookie.")
	}

	url, err := url.Parse(server)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse server url. %s", server)
	}
	url = url.JoinPath("app/preferences")

	var req *http.Request
	if req, err = http.NewRequest("GET", url.String(), nil); err != nil {
		return 0, fmt.Errorf("Failed to create request: %v", err)
	}
	if apikey != "" {
		req.Header.Set("Authorization", "Bearer "+apikey)
	} else {
		req.AddCookie(sessionCookie)
	}
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return 0, fmt.Errorf("Failed to get http response from endpoint Qbittorrent: %v", err)
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Got non-200 status code %d from Qbittorrent.", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("Error reading response body: %v", err)
	}

	var prefs qbPreferences
	if err = json.Unmarshal(body, &prefs); err != nil {
		return 0, fmt.Errorf("Failed to unmarshal response body from Qbittorrent: %v.", err)
	}

	return prefs.ListenPort, nil
}

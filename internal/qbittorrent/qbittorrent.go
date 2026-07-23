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

// the relevant qbittorrent Application preferences
type qbPreferences struct {
	ListenPort int `json:"listen_port"`
}

func Notify(server, apikey *string, port int) error {
	reqUrl, err := url.Parse(*server)
	if err != nil {
		return fmt.Errorf("Failed to parse server url. %s", *server)
	}
	reqUrl = reqUrl.JoinPath("app/setPreferences")

	prefs := qbPreferences{
		ListenPort: port,
	}
	encodedJSON, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("Failed to marshal JSON for preferences. Error: %v", err)
	}

	// qbittorrent wants the preference values encoded as form values.
	formValues := url.Values{}
	formValues.Add("json", string(encodedJSON))
	var req *http.Request
	if req, err = http.NewRequest("POST", reqUrl.String(), strings.NewReader(formValues.Encode())); err != nil {
		return fmt.Errorf("Failed to create request. Error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+(*apikey))
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return fmt.Errorf("Failed to get response from Qbittorrent. Error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Got non-200 status code %d from Qbittorrent", resp.StatusCode)
	}

	return nil
}

func Confirm(server, apikey *string) (int, error) {
	url, err := url.Parse(*server)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse server url. %s", *server)
	}
	url = url.JoinPath("app/preferences")

	var req *http.Request
	if req, err = http.NewRequest("GET", url.String(), nil); err != nil {
		return 0, fmt.Errorf("Failed to create request. Error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+(*apikey))
	req.Header.Set("User-Agent", "github.com/jdelkins/pia-tools")

	var resp *http.Response
	if resp, err = cleanhttp.DefaultPooledClient().Do(req); err != nil {
		return 0, fmt.Errorf("Failed to get http response from endpoint Qbittorrent. Error: %v", err)
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Got non-200 status code %d from Qbittorrent.", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("Error reading response body. Error: %v", err)
	}

	var prefs qbPreferences
	if err = json.Unmarshal(body, &prefs); err != nil {
		return 0, fmt.Errorf("Failed to unmarshal response body from Qbittorrent. Error: %v.", err)
	}

	return prefs.ListenPort, nil
}

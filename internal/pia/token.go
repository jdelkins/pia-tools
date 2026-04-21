package pia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Token struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

func (t Token) Valid() bool {
	return t.Token != "" && time.Now().Before(t.Expiry)
}

func (tun *Tunnel) NewToken(username string, password string) error {
	vals := url.Values{
		"username": {username},
		"password": {password},
	}
	req, err := http.NewRequest("POST", "https://www.privateinternetaccess.com/api/client/v2/token", strings.NewReader(vals.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Token   string `json:"token"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}
	if tokenResp.Token == "" {
		if tokenResp.Status != "" || tokenResp.Message != "" {
			return fmt.Errorf("Error generating PIA token: status=\"%s\" message=\"%s\"", tokenResp.Status, tokenResp.Message)
		}
		return fmt.Errorf("Error generating PIA token: empty token response (HTTP %s)", resp.Status)
	}
	day, _ := time.ParseDuration("23h55m")
	tun.Token = Token{
		Token:  tokenResp.Token,
		Expiry: time.Now().Add(day),
	}
	return nil
}

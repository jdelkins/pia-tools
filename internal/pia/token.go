package pia

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	url := fmt.Sprintf("https://%s/authv3/generateToken", tun.Region.MetaServer().Ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	resp, err := doRequest(req, tun.Region.MetaServer().Cn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var vals map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&vals); err != nil {
		return err
	}
	if vals["status"] != "OK" {
		return fmt.Errorf("Error generating PIA token: status=\"%s\" message=\"%s\"", vals["status"], vals["message"])
	}
	day, _ := time.ParseDuration("23h55m")
	tun.Token = Token{
		Token:  vals["token"],
		Expiry: time.Now().Add(day),
	}
	return nil
}

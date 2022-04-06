package pia

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PortForwardSig struct {
	Port      int       `json:"port"`
	Expiry    time.Time `json:"expires_at"`
	Signature string    `json:"signature"`
	Payload   string    `json:"payload"`
}

func (tun *Tunnel) NewPFSig() error {
	url := fmt.Sprintf("https://%s:19999/getSignature", tun.ServerVip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("token", tun.Token.Token)
	req.URL.RawQuery = q.Encode()
	resp, err := doRequest(req, tun.WgServer.Cn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var r struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		PortForwardSig
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if r.Status != "OK" {
		return fmt.Errorf("Could not get new port forward signature: status=\"%s\" message=\"%s\"", r.Status, r.Message)
	}
	payload_b, err := base64.StdEncoding.DecodeString(r.Payload)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payload_b, &r); err != nil {
		return err
	}
	tun.PFSig = r.PortForwardSig
	return nil
}

func (tun *Tunnel) BindPF() error {
	url := fmt.Sprintf("https://%s:19999/bindPort", tun.ServerVip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("payload", tun.PFSig.Payload)
	q.Add("signature", tun.PFSig.Signature)
	req.URL.RawQuery = q.Encode()
	resp, err := doRequest(req, tun.WgServer.Cn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var r struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if r.Status != "OK" {
		return fmt.Errorf("Could not bind port forward assignment: status=\"%s\" message=\"%s\"", r.Status, r.Message)
	}
	tun.Status = r.Status
	tun.Message = r.Message
	return nil
}

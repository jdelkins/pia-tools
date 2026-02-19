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
	// Explanation of PIA's /getSignature endpoint
	// You call this endpoint with a valid token, and it returns a json string
	// encoding a dict with these fields:
	// - status  : don't want to keep
	// - message : don't want to keep
	// - payload : want to keep
	// Payload is where the important info (especially the assigned port) is,
	// so we need to unpack it do do anything with it. Luckily, the payload
	// field is a base64-encoded json string as well, with (at least) these
	// fields:
	// - port       : want to keep
	// - expires_at : want to keep
	// - signature  : want to keep
	// I don't know exactly what "signature" is for, but we need to carry it to
	// refresh the assginment.
	// I use a kind of cute approach to store both the payload as well as the
	// decoded contents of payload, all in one little struct, namely to unmarshal
	// payload on top of its containing struct.
	url := fmt.Sprintf("https://%s:19999/getSignature", tun.ServerVip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("token", tun.Token.Token)
	req.URL.RawQuery = q.Encode()
	resp, err := doRequest(req, tun.Region.WgServer().Cn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// A disposable wrapper to grab the request status and any error message.
	// We won't keep this info, just the stuff in PortForwardSig. For now,
	// we'll be just getting PortForwardSig.Payload
	var r struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		PortForwardSig
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	if r.Status != "OK" {
		return fmt.Errorf("could not get new port forward signature: status=\"%s\" message=\"%s\"", r.Status, r.Message)
	}
	payload_b, err := base64.StdEncoding.DecodeString(r.Payload)
	if err != nil {
		return err
	}
	// Payload's contained fields will now be brothers of Payload
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
	resp, err := doRequest(req, tun.Region.WgServer().Cn)
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
		return fmt.Errorf("could not bind port forward assignment: status=\"%s\" message=\"%s\"", r.Status, r.Message)
	}
	tun.Status = r.Status
	tun.Message = r.Message
	return nil
}

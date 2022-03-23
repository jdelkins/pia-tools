package pia

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const path_cache = "/var/cache/private/"

type Server struct {
	Ip string `json:"ip"`
	Cn string `json:"cn"`
}

type Tunnel struct {
	MetaServer   Server
	WgServer     Server
	Status       string   `json:"status"`
	ServerPubkey string   `json:"server_key"`
	ServerPort   int      `json:"server_port"`
	ServerIp     string   `json:"server_ip"`
	ServerVip    string   `json:"server_vip"`
	PeerIp       string   `json:"peer_ip"`
	PrivateKey   string   `json:"peer_privkey"`
	PublicKey    string   `json:"peer_pubkey"`
	DnsServers   []string `json:"dns_servers"`
	Token        Token    `json:"token"`
	Message      string   `json:"message"`
	Interface    string   `json:"interface"`
}

const pia_url_servers = "https://serverlist.piaservers.net/vpninfo/servers/v4"

func do_request(req *http.Request) (*http.Response, error) {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetServers(region string) (*Tunnel, error) {
	req, err := http.NewRequest("GET", pia_url_servers, nil)
	if err != nil {
		return nil, err
	}
	resp, err := do_request(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tun Tunnel
	var res struct {
		Regions []struct {
			Id      string              `json:"id"`
			Servers map[string][]Server `json:"servers"`
		} `json:"regions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	for r := range res.Regions {
		if res.Regions[r].Id == region {
			tun.MetaServer = res.Regions[r].Servers["meta"][0]
			tun.WgServer = res.Regions[r].Servers["wg"][0]
			return &tun, nil
		}
	}
	return nil, fmt.Errorf("get_servers: Region %s not found", region)
}



func ActivateTunnel(tun *Tunnel) error {
	url := fmt.Sprintf("https://%s:1337/addKey", tun.WgServer.Ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("pt", tun.Token.Token)
	q.Add("pubkey", tun.PublicKey)
	req.URL.RawQuery = q.Encode()
	resp, err := do_request(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(tun); err != nil {
		return err
	}

	if tun.Status != "OK" {
		return fmt.Errorf("PIA returned the following error: %s: %s", tun.Status, tun.Message)
	}

	return nil
}

func SaveCache(tun *Tunnel) error {
	path := fmt.Sprintf("%s/PIA_%s.json", path_cache, tun.Interface)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(tun); err != nil {
		return err
	}
	return nil
}

func ReadCache(ifname string) (*Tunnel, error) {
	path := fmt.Sprintf("%s/PIA_%s.json", path_cache, ifname)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var tun Tunnel
	if err := json.NewDecoder(file).Decode(&tun); err != nil {
		return nil, err
	}
	return &tun, err
}

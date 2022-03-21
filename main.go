package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"text/template"
)

const region = "ca_toronto"
const pia_username = "***REMOVED***"
const pia_password = "***REMOVED***"
const pia_url_servers = "https://serverlist.piaservers.net/vpninfo/servers/v4"
const pia_url_token = "https://www.privateinternetaccess.com/api/client/v2/token"
const path_netdev = "/tmp/pia.netdev"
const path_network = "/tmp/pia.network"
const tmpl_netdev = `[NetDev]
Name=wg_pia
Kind=wireguard

[WireGuard]
PrivateKey={{ .PrivateKey }}
FirewallMark=12
#RouteTable=vpn

[WireGuardPeer]
PublicKey={{ .ServerPubkey }}
AllowedIPs=0.0.0.0/0
Endpoint={{ .ServerIp }}:{{ .ServerPort }}
PersistentKeepalive=25
#RouteTable=vpn
`
const tmpl_network = `[Match]
Name=wg_pia
Type=wireguard

[Network]
Address={{ .PeerIp }}/32

[Route]
Destination={{ .ServerVip }}/32
#Table=vpn
Scope=link

[Route]
Destination=0.0.0.0/0
Gateway={{ .ServerVip }}
GatewayOnLink=true
#Table=vpn
Scope=global
`

type Server struct {
	Ip string `json:"ip"`
	Cn string `json:"cn"`
}

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

func get_servers(region string) (*Server, *Server, error) {
	req, err := http.NewRequest("GET", pia_url_servers, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := do_request(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Regions []struct {
			Id      string              `json:"id"`
			Servers map[string][]Server `json:"servers"`
		} `json:"regions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, nil, err
	}

	for r := range res.Regions {
		if res.Regions[r].Id == region {
			return &res.Regions[r].Servers["meta"][0], &res.Regions[r].Servers["wg"][0], nil
		}
	}
	return nil, nil, fmt.Errorf("get_servers: Region %s not found", region)
}

func get_token(meta *Server, username string, password string) (string, error) {
	url := fmt.Sprintf("https://%s/authv3/generateToken", meta.Ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(username, password)
	resp, err := do_request(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var vals map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&vals); err != nil {
		return "", err
	}
	return vals["token"], nil
}

func activate_tunnel(server *Server, token, privkey, pubkey string) (string, error) {
	url := fmt.Sprintf("https://%s:1337/addKey", server.Ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("pt", token)
	q.Add("pubkey", pubkey)
	req.URL.RawQuery = q.Encode()
	resp, err := do_request(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var server_info struct {
		Status       string   `json:"status"`
		ServerPubkey string   `json:"server_key"`
		ServerPort   int      `json:"server_port"`
		ServerIp     string   `json:"server_ip"`
		ServerVip    string   `json:"server_vip"`
		PeerIp       string   `json:"peer_ip"`
		PrivateKey   string   `json:"peer_privkey"`
		PublicKey    string   `json:"peer_pubkey"`
		DnsServers   []string `json:"dns_servers"`
	}
	// Returns this stuff:
	// 2022/03/16 00:59:22 Server info: {
	//     "status": "OK",
	//     "server_key": "WfzI/GRGdwIRefbYas7II+/zOKJMH/lifQsTyygT1Wk=",
	//     "server_port": 1337,
	//     "server_ip": "66.115.142.46",
	//     "server_vip": "10.44.128.1",
	//     "peer_ip": "10.44.229.219",
	//     "peer_pubkey": "1VBJGvDK3RmY6hH0j2jaN8dOAze1wbiQ2frjQvVo4jw=",
	//     "dns_servers": [
	//         "10.0.0.243",
	//         "10.0.0.242"
	//     ]
	// }
	if err := json.NewDecoder(resp.Body).Decode(&server_info); err != nil {
		return "", err
	}
	server_info.PrivateKey = privkey

	fmt.Printf("%+v\n", server_info)

	if server_info.Status != "OK" {
		return "", fmt.Errorf("PIA returned the following error: %s", server_info.Status)
	}

	// Generate .netdev
	nd_tmpl, err := template.New("netdev").Parse(tmpl_netdev)
	if err != nil {
		return "", err
	}
	log.Printf("Generating netdev file %s", path_netdev)
	nd, err := os.Create(path_netdev)
	err = nd_tmpl.Execute(nd, server_info)
	if err != nil {
		return "", err
	}

	// Generate .netdev
	nw_tmpl, err := template.New("network").Parse(tmpl_network)
	if err != nil {
		return "", err
	}
	log.Printf("Generating network file %s", path_network)
	nw, err := os.Create(path_network)
	err = nw_tmpl.Execute(nw, server_info)
	if err != nil {
		return "", err
	}

	return "Done", nil
}

func gen_keypair() (privkey string, pubkey string, err error) {
	privkey_b, err := exec.Command("wg", "genkey").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("%v; %s", err, privkey_b)
	}
	privkey = string(privkey_b)

	cmd := exec.Command("wg", "pubkey")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", "", fmt.Errorf("StdinPipe: %v", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, privkey)
	}()
	pubkey_b, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("%v; %s", err, pubkey_b)
	}
	pubkey = string(pubkey_b)

	return
}

func main() {
	privkey, pubkey, err := gen_keypair()
	if err != nil {
		log.Fatalf("Could not generate keypair: %v", err)
	}
	log.Printf("Privkey = %s, Pubkey = %s", privkey, pubkey)

	meta, wg, err := get_servers(region)
	if err != nil {
		log.Fatalf("Could not determine servers: %v", err)
	}
	log.Printf("meta = %s, wg = %s", meta.Ip, wg.Ip)
	token, err := get_token(meta, pia_username, pia_password)
	if err != nil {
		log.Fatalf("Could not get token: %v", err)
	}
	log.Printf("token = %s", token)
	msg, err := activate_tunnel(wg, token, privkey, pubkey)
	if err != nil {
		log.Fatalf("Could not register public key: %v", err)
	}
	log.Printf("Server info: %s", msg)
}

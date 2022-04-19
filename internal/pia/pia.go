package pia

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
)

const pathCache = "/var/cache/pia"

type Server struct {
	Ip string `json:"ip"`
	Cn string `json:"cn"`
}

type Tunnel struct {
	Region       Region
	Status       string         `json:"status"`
	ServerPubkey string         `json:"server_key"`
	ServerPort   int            `json:"server_port"`
	ServerIp     string         `json:"server_ip"`
	ServerVip    string         `json:"server_vip"`
	PeerIp       string         `json:"peer_ip"`
	PrivateKey   string         `json:"peer_privkey"`
	PublicKey    string         `json:"peer_pubkey"`
	DnsServers   []string       `json:"dns_servers"`
	Token        Token          `json:"token"`
	Message      string         `json:"message"`
	Interface    string         `json:"interface"`
	PFSig        PortForwardSig `json:",omitempty"`
}

func NewTunnel(region *Region, intf string) *Tunnel {
	return &Tunnel{Region: *region, Interface: intf}
}

var _piaCertpool *x509.CertPool = nil

func getPiaCertpool() *x509.CertPool {
	if _piaCertpool == nil {
		p, _ := pem.Decode([]byte(`
-----BEGIN CERTIFICATE-----
MIIHqzCCBZOgAwIBAgIJAJ0u+vODZJntMA0GCSqGSIb3DQEBDQUAMIHoMQswCQYD
VQQGEwJVUzELMAkGA1UECBMCQ0ExEzARBgNVBAcTCkxvc0FuZ2VsZXMxIDAeBgNV
BAoTF1ByaXZhdGUgSW50ZXJuZXQgQWNjZXNzMSAwHgYDVQQLExdQcml2YXRlIElu
dGVybmV0IEFjY2VzczEgMB4GA1UEAxMXUHJpdmF0ZSBJbnRlcm5ldCBBY2Nlc3Mx
IDAeBgNVBCkTF1ByaXZhdGUgSW50ZXJuZXQgQWNjZXNzMS8wLQYJKoZIhvcNAQkB
FiBzZWN1cmVAcHJpdmF0ZWludGVybmV0YWNjZXNzLmNvbTAeFw0xNDA0MTcxNzQw
MzNaFw0zNDA0MTIxNzQwMzNaMIHoMQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0Ex
EzARBgNVBAcTCkxvc0FuZ2VsZXMxIDAeBgNVBAoTF1ByaXZhdGUgSW50ZXJuZXQg
QWNjZXNzMSAwHgYDVQQLExdQcml2YXRlIEludGVybmV0IEFjY2VzczEgMB4GA1UE
AxMXUHJpdmF0ZSBJbnRlcm5ldCBBY2Nlc3MxIDAeBgNVBCkTF1ByaXZhdGUgSW50
ZXJuZXQgQWNjZXNzMS8wLQYJKoZIhvcNAQkBFiBzZWN1cmVAcHJpdmF0ZWludGVy
bmV0YWNjZXNzLmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBALVk
hjumaqBbL8aSgj6xbX1QPTfTd1qHsAZd2B97m8Vw31c/2yQgZNf5qZY0+jOIHULN
De4R9TIvyBEbvnAg/OkPw8n/+ScgYOeH876VUXzjLDBnDb8DLr/+w9oVsuDeFJ9K
V2UFM1OYX0SnkHnrYAN2QLF98ESK4NCSU01h5zkcgmQ+qKSfA9Ny0/UpsKPBFqsQ
25NvjDWFhCpeqCHKUJ4Be27CDbSl7lAkBuHMPHJs8f8xPgAbHRXZOxVCpayZ2SND
fCwsnGWpWFoMGvdMbygngCn6jA/W1VSFOlRlfLuuGe7QFfDwA0jaLCxuWt/BgZyl
p7tAzYKR8lnWmtUCPm4+BtjyVDYtDCiGBD9Z4P13RFWvJHw5aapx/5W/CuvVyI7p
Kwvc2IT+KPxCUhH1XI8ca5RN3C9NoPJJf6qpg4g0rJH3aaWkoMRrYvQ+5PXXYUzj
tRHImghRGd/ydERYoAZXuGSbPkm9Y/p2X8unLcW+F0xpJD98+ZI+tzSsI99Zs5wi
jSUGYr9/j18KHFTMQ8n+1jauc5bCCegN27dPeKXNSZ5riXFL2XX6BkY68y58UaNz
meGMiUL9BOV1iV+PMb7B7PYs7oFLjAhh0EdyvfHkrh/ZV9BEhtFa7yXp8XR0J6vz
1YV9R6DYJmLjOEbhU8N0gc3tZm4Qz39lIIG6w3FDAgMBAAGjggFUMIIBUDAdBgNV
HQ4EFgQUrsRtyWJftjpdRM0+925Y6Cl08SUwggEfBgNVHSMEggEWMIIBEoAUrsRt
yWJftjpdRM0+925Y6Cl08SWhge6kgeswgegxCzAJBgNVBAYTAlVTMQswCQYDVQQI
EwJDQTETMBEGA1UEBxMKTG9zQW5nZWxlczEgMB4GA1UEChMXUHJpdmF0ZSBJbnRl
cm5ldCBBY2Nlc3MxIDAeBgNVBAsTF1ByaXZhdGUgSW50ZXJuZXQgQWNjZXNzMSAw
HgYDVQQDExdQcml2YXRlIEludGVybmV0IEFjY2VzczEgMB4GA1UEKRMXUHJpdmF0
ZSBJbnRlcm5ldCBBY2Nlc3MxLzAtBgkqhkiG9w0BCQEWIHNlY3VyZUBwcml2YXRl
aW50ZXJuZXRhY2Nlc3MuY29tggkAnS7684Nkme0wDAYDVR0TBAUwAwEB/zANBgkq
hkiG9w0BAQ0FAAOCAgEAJsfhsPk3r8kLXLxY+v+vHzbr4ufNtqnL9/1Uuf8NrsCt
pXAoyZ0YqfbkWx3NHTZ7OE9ZRhdMP/RqHQE1p4N4Sa1nZKhTKasV6KhHDqSCt/dv
Em89xWm2MVA7nyzQxVlHa9AkcBaemcXEiyT19XdpiXOP4Vhs+J1R5m8zQOxZlV1G
tF9vsXmJqWZpOVPmZ8f35BCsYPvv4yMewnrtAC8PFEK/bOPeYcKN50bol22QYaZu
LfpkHfNiFTnfMh8sl/ablPyNY7DUNiP5DRcMdIwmfGQxR5WEQoHL3yPJ42LkB5zs
6jIm26DGNXfwura/mi105+ENH1CaROtRYwkiHb08U6qLXXJz80mWJkT90nr8Asj3
5xN2cUppg74nG3YVav/38P48T56hG1NHbYF5uOCske19F6wi9maUoto/3vEr0rnX
JUp2KODmKdvBI7co245lHBABWikk8VfejQSlCtDBXn644ZMtAdoxKNfR2WTFVEwJ
iyd1Fzx0yujuiXDROLhISLQDRjVVAvawrAtLZWYK31bY7KlezPlQnl/D9Asxe85l
8jO5+0LdJ6VyOs/Hd4w52alDW/MFySDZSfQHMTIc30hLBJ8OnCEIvluVQQ2UQvoW
+no177N9L2Y+M9TcTA62ZyMXShHQGeh20rb4kK8f+iFX8NxtdHVSkxMEFSfDDyQ=
-----END CERTIFICATE-----
		`))
		ca, _ := x509.ParseCertificate(p.Bytes)
		_piaCertpool = x509.NewCertPool()
		_piaCertpool.AddCert(ca)
	}
	return _piaCertpool
}

func doRequest(req *http.Request, server_name string) (*http.Response, error) {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: server_name,
				RootCAs:    getPiaCertpool(),
			},
		},
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (tun *Tunnel) Activate() error {
	url := fmt.Sprintf("https://%s:1337/addKey", tun.Region.WgServer().Ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("pt", tun.Token.Token)
	q.Add("pubkey", tun.PublicKey)
	req.URL.RawQuery = q.Encode()
	resp, err := doRequest(req, tun.Region.WgServer().Cn)
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

func (tun *Tunnel) SaveCache() error {
	path := fmt.Sprintf("%s/%s.json", pathCache, tun.Interface)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o660)
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
	path := fmt.Sprintf("%s/%s.json", pathCache, ifname)
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

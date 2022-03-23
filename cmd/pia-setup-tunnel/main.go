package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/jdelkins/pia-tools/internal/fileops"
	"github.com/jdelkins/pia-tools/internal/pia"
)

// const region = "ca_toronto"
// const pia_username = "***REMOVED***"
// const pia_password = "***REMOVED***"
var (
	path_netdev       string
	path_network      string
	path_netdev_tmpl  string
	path_network_tmpl string
	pia_username      string
	pia_password      string
	region            string
	wg_if             string
)

func gen_keypair(tun *pia.Tunnel) error {
	privkey_b, err := exec.Command("wg", "genkey").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v; %s", err, privkey_b)
	}
	tun.PrivateKey = strings.TrimSpace(string(privkey_b))

	cmd := exec.Command("wg", "pubkey")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("StdinPipe: %v", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, tun.PrivateKey)
	}()
	pubkey_b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v; %s", err, pubkey_b)
	}
	tun.PublicKey = strings.TrimSpace(string(pubkey_b))

	return nil
}

func parse_args() error {
	const path_systemd_networkd = "/etc/systemd/network"
	flag.StringVar(&wg_if, "ifname", "pia", "name of interface \"IF\", where the systemd-networkd files will be called /etc/systemd/network/IF.{netdev,network}")
	flag.StringVar(&pia_username, "username", "", "PIA username")
	flag.StringVar(&pia_password, "password", "", "PIA password")
	flag.StringVar(&region, "region", "ca_toronto", "PIA region id")
	flag.Parse()
	if pia_username == "" || pia_password == "" {
		return fmt.Errorf("Username and/or password were not provided")
	}
	path_netdev = fmt.Sprintf("%s/%s.netdev", path_systemd_networkd, wg_if)
	path_network = fmt.Sprintf("%s/%s.network", path_systemd_networkd, wg_if)
	path_netdev_tmpl = fmt.Sprintf("%s/%s.netdev.tmpl", path_systemd_networkd, wg_if)
	path_network_tmpl = fmt.Sprintf("%s/%s.network.tmpl", path_systemd_networkd, wg_if)
	return nil
}

func main() {
	if err := parse_args(); err != nil {
		flag.Usage()
		log.Fatalf("%v", err)
	}
	tun, err := pia.GetServers(region)
	if err != nil {
		log.Fatalf("Could not determine servers: %v", err)
	}
	tun.Interface = wg_if
	if err = gen_keypair(tun); err != nil {
		log.Fatalf("Could not generate keypair: %v", err)
	}
	if err := pia.GetToken(tun, pia_username, pia_password); err != nil {
		log.Fatalf("Could not get token: %v", err)
	}
	if err := pia.ActivateTunnel(tun); err != nil {
		log.Fatalf("Could not register public key: %v", err)
	}
	if err := fileops.CreateNetdevFile(tun, path_netdev, path_netdev_tmpl); err != nil {
		log.Fatalf("Could not create %s file: %v", path_netdev, err)
	}
	if err := fileops.CreateNetworkFile(tun, path_network, path_network_tmpl); err != nil {
		log.Fatalf("Could not create %s file: %v", path_network, err)
	}
}

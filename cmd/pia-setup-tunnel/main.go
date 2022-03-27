package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/jdelkins/pia-tools/internal/fileops"
	"github.com/jdelkins/pia-tools/internal/pia"
)

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

func parse_args() error {
	const path_sn = "/etc/systemd/network"
	flag.StringVar(&wg_if, "ifname", "pia", "name of interface \"IF\", where the systemd-networkd files will be called /etc/systemd/network/IF.{netdev,network}")
	flag.StringVar(&pia_username, "username", "", "PIA username")
	flag.StringVar(&pia_password, "password", "", "PIA password")
	flag.StringVar(&region, "region", "ca_toronto", "PIA region id")
	flag.Parse()
	if pia_username == "" || pia_password == "" {
		return fmt.Errorf("Username and/or password were not provided")
	}
	path_netdev = fmt.Sprintf("%s/%s.netdev", path_sn, wg_if)
	path_network = fmt.Sprintf("%s/%s.network", path_sn, wg_if)
	path_netdev_tmpl = fmt.Sprintf("%s/%s.netdev.tmpl", path_sn, wg_if)
	path_network_tmpl = fmt.Sprintf("%s/%s.network.tmpl", path_sn, wg_if)
	return nil
}

func main() {
	if err := parse_args(); err != nil {
		flag.Usage()
		log.Fatalf("%v", err)
	}
	tun, err := pia.GetServers(region)
	if err != nil {
		log.Fatalf("Could not get server list: %v", err)
	}
	tun.Interface = wg_if
	if err = gen_keypair(tun); err != nil {
		log.Fatalf("Could not generate keypair: %v", err)
	}
	if !tun.Token.Valid() {
		if err := tun.NewToken(pia_username, pia_password); err != nil {
			log.Fatalf("Could not get token: %v", err)
		}
	}
	if err := tun.Activate(); err != nil {
		log.Fatalf("Could not register public key: %v", err)
	}
	if err := fileops.CreateNetdevFile(tun, path_netdev, path_netdev_tmpl); err != nil {
		log.Fatalf("Could not create %s file: %v", path_netdev, err)
	}
	if err := fileops.CreateNetworkFile(tun, path_network, path_network_tmpl); err != nil {
		log.Fatalf("Could not create %s file: %v", path_network, err)
	}
	if err := tun.SaveCache(); err != nil {
		log.Fatalf("Could not save cache: %v", err)
	}
	fmt.Println(tun.Status)
}

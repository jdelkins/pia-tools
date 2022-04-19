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

func parseArgs() error {
	const path_sn = "/etc/systemd/network"
	flag.StringVar(&wg_if, "ifname", "pia", "name of interface \"IF\", where the systemd-networkd files will be called /etc/systemd/network/IF.{netdev,network}")
	flag.StringVar(&pia_username, "username", "", "PIA username")
	flag.StringVar(&pia_password, "password", "", "PIA password")
	flag.StringVar(&region, "region", "auto", "PIA region id")
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
	if err := parseArgs(); err != nil {
		flag.Usage()
		log.Panicf("%v", err)
	}

	// Find the "best" region if requested
	if region == "auto" || region == "" {
		regions, err := pia.RegionsWithPingTime()
		if err != nil {
			log.Panicf("Could not enumerate regions: %v", err)
		}
		// region list should be sorted by ping time from best to worst, so we
		// just need to find the first one in the list that has both port
		// forwarding and wireguard
		for i := range regions {
			r := &regions[i]
			if r.HasWg() && r.PortForward {
				region = r.Id
				fmt.Printf("Selected region %s (%s), having ping time %d ms\n", r.Id, r.Name, r.PingTime.Milliseconds())
				break
			}
		}
	}

	// Make sure the provided region supports WG
	reg, err := pia.FindRegion(region)
	if err != nil {
		log.Panicf("%v", err)
	}
	if !reg.HasWg() {
		log.Panicf("Region %s (%s) does not support WireGuard at this time", reg.Id, reg.Name)
	}

	// Create a Tunnel struct and populate it with fresh WG keys and an access token
	tun := pia.NewTunnel(reg, wg_if)
	defer func() {
		if err := tun.SaveCache(); err != nil {
			log.Panicf("Could not save cache: %v", err)
		}
	}()
	if err = genKeypair(tun); err != nil {
		log.Panicf("Could not generate keypair: %v", err)
	}
	if !tun.Token.Valid() {
		if err := tun.NewToken(pia_username, pia_password); err != nil {
			log.Panicf("Could not get token: %v", err)
		}
	}

	// Register the WG keys
	if err := tun.Activate(); err != nil {
		log.Panicf("Could not register public key: %v", err)
	}

	// Finally, populate the templates
	if err := fileops.CreateNetdevFile(tun, path_netdev, path_netdev_tmpl); err != nil {
		log.Panicf("Could not create %s file: %v", path_netdev, err)
	}
	if err := fileops.CreateNetworkFile(tun, path_network, path_network_tmpl); err != nil {
		log.Panicf("Could not create %s file: %v", path_network, err)
	}
	fmt.Println(tun.Status)
}

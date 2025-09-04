package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jdelkins/pia-tools/internal/fileops"
	"github.com/jdelkins/pia-tools/internal/pia"
	flag "github.com/spf13/pflag"
)

var (
	path_netdev       string
	path_network      string
	path_netdev_tmpl  string
	path_network_tmpl string
	pia_username      string
	pia_password      string
	reg_id            string
	wg_if             string
)

func init() {
	const path_sn = "/etc/systemd/network"
	flag.StringVarP(&wg_if, "ifname", "i", "pia", "name of interface \"IF\", where the systemd-networkd files will be called /etc/systemd/network/IF.{netdev,network}")
	flag.StringVarP(&pia_username, "username", "u", os.Getenv("PIA_USERNAME"), "PIA username (REQUIRED)")
	flag.StringVarP(&pia_password, "password", "p", os.Getenv("PIA_PASSWORD"), "PIA password (REQUIRED)")
	flag.StringVarP(&reg_id, "region", "r", "auto", "PIA region id")
	flag.StringVarP(&path_netdev, "netdev", "n", "", "Path to generated netdev unit file (see systemd.netdev(5))")
	flag.StringVarP(&path_network, "network", "N", "", "Path to generated network unit file (see systemd.network(5))")
	flag.StringVarP(&path_netdev_tmpl,  "netdev-template", "t", "", "Path to netdev template unit file (see systemd.netdev(5))")
	flag.StringVarP(&path_network_tmpl, "network-template", "T", "", "Path to network template unit file (see systemd.network(5))")
	flag.Parse()
	if path_netdev == "" {
		path_netdev = fmt.Sprintf("%s/%s.netdev", path_sn, wg_if)
	}
	if path_network == "" {
		path_network = fmt.Sprintf("%s/%s.network", path_sn, wg_if)
	}
	if path_netdev_tmpl == "" {
		path_netdev_tmpl = fmt.Sprintf("%s/%s.netdev.tmpl", path_sn, wg_if)
	}
	if path_network_tmpl == "" {
		path_network_tmpl = fmt.Sprintf("%s/%s.network.tmpl", path_sn, wg_if)
	}
	if pia_username == "" || pia_password == "" {
		fmt.Fprintf(os.Stderr, "%s: --username and --password are required arguments. Aborting.\n\n", os.Args[0])
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	// Find the "best" reg_id if requested
	var reg *pia.Region
	if reg_id == "auto" || reg_id == "" {
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
				reg = r
				fmt.Printf("Selected region %s (%s), having ping time %d ms\n", r.Id, r.Name, r.PingTime.Milliseconds())
				break
			}
		}
	}

	// Get configured region details, if not "auto"
	if reg == nil {
		var err error
		reg, err = pia.FindRegion(reg_id)
		if err != nil {
			log.Panicf("%v", err)
		}
	}

	// Create a Tunnel struct and populate it with fresh WG keys and an access token
	tun := pia.NewTunnel(reg, wg_if)
	defer func() {
		if err := tun.SaveCache(); err != nil {
			log.Panicf("Could not save cache: %v", err)
		}
	}()
	if err := genKeypair(tun); err != nil {
		log.Panicf("Could not generate keypair: %v", err)
	}
	if !tun.Token.Valid() {
		if err := tun.NewToken(pia_username, pia_password); err != nil {
			log.Panicf("Could not get token: %v", err)
		}
	}

	// Register the WG keys to our account (identified by access token)
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

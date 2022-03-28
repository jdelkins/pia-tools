package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/jdelkins/pia-tools/internal/pia"
	"github.com/jdelkins/pia-tools/internal/rtorrent"
)

var (
	pia_username string
	pia_password string
	wg_if        string
	refresh_only bool
	rtorrent_url string
)

func parse_args() {
	const path_systemd_networkd = "/etc/systemd/network"
	flag.StringVar(&wg_if, "ifname", "pia", "name of wireguard interface, used to determine cache filename")
	flag.StringVar(&pia_username, "username", "", "PIA username")
	flag.StringVar(&pia_password, "password", "", "PIA password")
	flag.BoolVar(&refresh_only, "refresh", false, "Refresh active (cached) port assignment")
	flag.StringVar(&rtorrent_url, "rtorrent", "", "Notify rtorrent of the assigned port via XML-RPC at this URL")
	flag.Parse()
}

func verify_port(tun *pia.Tunnel) int {
	port, err := rtorrent.Confirm(rtorrent_url)
	if err != nil {
		log.Fatalf("Could not verify rtorrent port: %v", err)
	}
	if port != tun.PFSig.Port {
		log.Fatalf("PIA assigned us port %d, but rtorrent reports port is %d", tun.PFSig.Port, port)
	}
	return port
}

func main() {
	parse_args()
	port_verified := false
	tun, err := pia.ReadCache(wg_if)
	if err != nil {
		log.Fatalf("Could not read cache: %v", err)
	}
	if !tun.Token.Valid() {
		if pia_username == "" || pia_password == "" {
			log.Fatalf("Token expired and user/pass not provided")
		}
		if err := tun.NewToken(pia_username, pia_password); err != nil {
			log.Fatalf("Token expired; error refreshing: %v", err)
		}
	}
	if !refresh_only {
		if err := tun.NewPFSig(); err != nil {
			log.Fatalf("Could not get port forwarding signature: %v", err)
		}
		if rtorrent_url != "" {
			if err := rtorrent.Notify(rtorrent_url, tun.PFSig.Port); err != nil {
				log.Fatalf("Could not notify rtorrent (at %s) of assigned port: %v", rtorrent_url, err)
			}
			verify_port(tun)
			port_verified = true
		}
		defer func() {
			if err := tun.SaveCache(); err != nil {
				log.Fatalf("Could not save cache: %v", err)
			}
		}()
	}
	if rtorrent_url != "" && !port_verified {
		verify_port(tun)
		port_verified = true
	}
	if err := tun.BindPF(); err != nil {
		log.Fatalf("Could not pind port forwarding assignment: %v", err)
	}
	fmt.Printf("%s: %s (Port = %d)\n", tun.Status, tun.Message, tun.PFSig.Port)
}

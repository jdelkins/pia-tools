package main

import (
	"fmt"
	"log"

	"github.com/jdelkins/pia-tools/internal/pia"
	"github.com/jdelkins/pia-tools/internal/rtorrent"
	"github.com/jdelkins/pia-tools/internal/transmission"
	flag "github.com/spf13/pflag"
)

var (
	piaUsername          string
	piaPassword          string
	wg_if                string
	refreshOnly          bool
	rtorrentUrl          string
	transmissionAddress  string
	transmissionUsername string
	transmissionPassword string
)

func init() {
	flag.StringVarP(&wg_if, "ifname", "i", "pia", "name of wireguard interface, used to determine cache filename")
	flag.StringVarP(&piaUsername, "username", "u", "", "PIA username")
	flag.StringVarP(&piaPassword, "password", "p", "", "PIA password")
	flag.BoolVarP(&refreshOnly, "refresh", "r", false, "Refresh cached port assignment, rather than getting a new assignment")
	flag.StringVar(&rtorrentUrl, "rtorrent", "", "Notify rtorrent at this XML-RPC URL of the assigned port")
	flag.StringVar(&transmissionAddress, "transmission", "", "Notify transmission bittorrent server at this IP address of the asisgned port")
	flag.StringVar(&transmissionUsername, "transmission-username", "", "Transmission server username")
	flag.StringVar(&transmissionUsername, "transmission-password", "", "Transmission server password")
	flag.Parse()
}

func main() {
	tun, err := pia.ReadCache(wg_if)
	defer func() {
		if err := tun.SaveCache(); err != nil {
			log.Panicf("Could not save cache: %v", err)
		}
	}()
	if err != nil {
		log.Panicf("Could not read cache: %v", err)
	}
	if !tun.Token.Valid() {
		if piaUsername == "" || piaPassword == "" {
			log.Panicf("Token expired and user/pass not provided")
		}
		if err := tun.NewToken(piaUsername, piaPassword); err != nil {
			log.Panicf("Token expired; error refreshing: %v", err)
		}
	}
	if !refreshOnly {
		if err := tun.NewPFSig(); err != nil {
			log.Panicf("Could not get port forwarding signature: %v", err)
		}
	}
	if err := tun.BindPF(); err != nil {
		log.Panicf("Could not bind port forwarding assignment: %v", err)
	}
	if rtorrentUrl != "" {
		if err := rtorrent.Notify(rtorrentUrl, tun.PFSig.Port); err != nil {
			log.Panicf("Could not notify rtorrent (at %s) of assigned port: %v", rtorrentUrl, err)
		}
		port, err := rtorrent.Confirm(rtorrentUrl)
		if err != nil {
			log.Panicf("Could not verify rtorrent port: %v", err)
		}
		if port != tun.PFSig.Port {
			log.Panicf("PIA assigned us port %d, but rtorrent reports port is %d", tun.PFSig.Port, port)
		}
	}
	if transmissionAddress != "" {
		if err := transmission.Notify(transmissionAddress, transmissionUsername, transmissionPassword, tun.PFSig.Port); err != nil {
			log.Panicf("Could not notify transmission (at %s) of assigned port: %v", rtorrentUrl, err)
		}
		port, err := transmission.Confirm(transmissionAddress, transmissionUsername, transmissionPassword)
		if err != nil {
			log.Panicf("Could not verify transmission port: %v", err)
		}
		if port != tun.PFSig.Port {
			log.Panicf("PIA assigned us port %d, but transmission reports port is %d", tun.PFSig.Port, port)
		}
	}
	fmt.Printf("%s: %s (Port = %d)\n", tun.Status, tun.Message, tun.PFSig.Port)
}

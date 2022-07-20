package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/jdelkins/pia-tools/internal/pia"
	"github.com/jdelkins/pia-tools/internal/rtorrent"
	"github.com/jdelkins/pia-tools/internal/transmission"
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

func parseArgs() {
	flag.StringVar(&wg_if, "ifname", "pia", "name of wireguard interface, used to determine cache filename")
	flag.StringVar(&piaUsername, "username", "", "PIA username")
	flag.StringVar(&piaPassword, "password", "", "PIA password")
	flag.BoolVar(&refreshOnly, "refresh", false, "Refresh cached port assignment, rather than getting a new assignment")
	flag.StringVar(&rtorrentUrl, "rtorrent", "", "Notify rtorrent of the assigned port via XML-RPC at this URL")
	flag.StringVar(&transmissionAddress, "transmission", "", "Notify transmission bittorrent server at this address of the assigned port")
	flag.StringVar(&transmissionUsername, "transmission-username", "", "Transmission server username")
	flag.StringVar(&transmissionUsername, "transmission-password", "", "Transmission server password")
	flag.Parse()
}

func main() {
	parseArgs()
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

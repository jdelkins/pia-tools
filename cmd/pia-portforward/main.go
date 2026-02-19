package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/jdelkins/pia-tools/internal/pia"
	"github.com/jdelkins/pia-tools/internal/rtorrent"
	"github.com/jdelkins/pia-tools/internal/transmission"
)

type CLI struct {
	IfName   string `short:"i" aliases:"ifname" default:"pia" help:"Name of WireGuard interface, used to determine cache filename."`
	Username string `short:"u" name:"username" env:"PIA_USERNAME" help:"PIA username (required if token expired)."`
	Password string `short:"p" name:"password" env:"PIA_PASSWORD" help:"PIA password (required if token expired)."`

	Refresh bool `short:"r" name:"refresh" help:"Refresh cached port assignment rather than requesting a new one."`

	Rtorrent      string `name:"rtorrent" env:"RTORRENT" help:"XML-RPC URL of rtorrent server (for port forward notifications)."`
	Transmission  string `name:"transmission" env:"TRANSMISSION" help:"URL of transmission server RPC endpoint (for port forward notifications)."`
	TransUser     string `name:"transmission-username" env:"TRANSMISSION_USERNAME" help:"Transmission server username."`
	TransPassword string `name:"transmission-password" env:"TRANSMISSION_PASSWORD" help:"Transmission server password."`

	CacheDir string `short:"c" aliases:"cachedir" default:"/var/cache/pia" help:"Directory in which to store security-sensitive cache files."`
}

func main() {
	var cli CLI
	kong.Parse(&cli, kong.Name("pia-portforward"))

	// grab the cached tunnel info
	tun, err := pia.ReadCache(cli.CacheDir, cli.IfName)
	if err != nil {
		log.Panicf("Could not read cache: %v", err)
	}
	defer func() {
		if err := tun.SaveCache(cli.CacheDir); err != nil {
			log.Panicf("Could not save cache: %v", err)
		}
	}()

	// ensure our token is still valid, if not grab a new one
	if !tun.Token.Valid() {
		if cli.Username == "" || cli.Password == "" {
			log.Panicf("Token expired and user/pass not provided")
		}
		if err := tun.NewToken(cli.Username, cli.Password); err != nil {
			log.Panicf("Token expired; error refreshing: %v", err)
		}
	}

	// request new port unless --refresh
	if !cli.Refresh {
		if err := tun.NewPFSig(); err != nil {
			log.Panicf("Could not get port forwarding signature: %v", err)
		}
	}

	// bind the port to our virtual IP. If already active, effectuates the refresh
	if err := tun.BindPF(); err != nil {
		log.Panicf("Could not bind port forwarding assignment: %v", err)
	}

	// notify rtorrent
	if cli.Rtorrent != "" {
		if err := rtorrent.Notify(cli.Rtorrent, tun.PFSig.Port); err != nil {
			log.Panicf("Could not notify rtorrent (at %s) of assigned port: %v", cli.Rtorrent, err)
		}
		port, err := rtorrent.Confirm(cli.Rtorrent)
		if err != nil {
			log.Panicf("Could not verify rtorrent port: %v", err)
		}
		if port != tun.PFSig.Port {
			log.Panicf("PIA assigned us port %d, but rtorrent reports port is %d", tun.PFSig.Port, port)
		}
	}

	// notify transmission
	if cli.Transmission != "" {
		if err := transmission.Notify(cli.Transmission, cli.TransUser, cli.TransPassword, tun.PFSig.Port); err != nil {
			log.Panicf("Could not notify transmission (at %s) of assigned port: %v", cli.Transmission, err)
		}
		port, err := transmission.Confirm(cli.Transmission, cli.TransUser, cli.TransPassword)
		if err != nil {
			log.Panicf("Could not verify transmission port: %v", err)
		}
		if port != tun.PFSig.Port {
			log.Panicf("PIA assigned us port %d, but transmission reports port is %d", tun.PFSig.Port, port)
		}
	}

	// success!
	fmt.Printf("%s: %s (Port = %d)\n", tun.Status, tun.Message, tun.PFSig.Port)
}

package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/jdelkins/pia-tools/internal/fileops"
	"github.com/jdelkins/pia-tools/internal/pia"
)

type CLI struct {
	IfName    string `short:"i" aliases:"ifname" default:"pia" help:"Name of interface IF; default output/template paths derive from IF under /etc/systemd/network."`
	Username  string `short:"u" env:"PIA_USERNAME" required:"" help:"PIA username (required; may also be set via PIA_USERNAME)."`
	Password  string `short:"p" env:"PIA_PASSWORD" required:"" help:"PIA password (required; may also be set via PIA_PASSWORD)."`
	Region    string `short:"r" env:"PIA_REGION" default:"auto" help:"PIA region id (or 'auto')."`
	CacheDir  string `short:"c" aliases:"cachedir" default:"/var/cache/pia" help:"Path in which to store security-sensitive cache files."`
	WGBinary  string `short:"b" default:"wg" help:"Path to the 'wg' binary from wireguard-tools."`
	FromCache bool   `aliases:"cached" help:"Generate systemd-networkd files from the cached tunnel info."`

	// Comma-separated key/value spec parsed into a map by Kong.
	// Example:
	//   --netdev-file=output=/etc/systemd/network/pia.netdev,template=/etc/systemd/network/pia.netdev.tmpl,mode=0440,owner=fred,group=systemd-network
	NetdevFile  map[string]string `name:"netdev-file" mapsep:"," sep:"=" help:"File spec for generating the .netdev file (comma-separated key=value pairs). Keys: output,template,mode,owner,group"`
	NetworkFile map[string]string `name:"network-file" mapsep:"," sep:"=" help:"File spec for generating the .network file (comma-separated key=value pairs). Keys: output,template,mode,owner,group"`
}

func (c *CLI) AfterApply(ctx *kong.Context) error {
	_ = ctx
	const pathSN = "/etc/systemd/network"

	// Provide sane defaults if the flags are omitted or missing keys.
	if c.NetdevFile == nil {
		c.NetdevFile = map[string]string{}
	}
	if c.NetworkFile == nil {
		c.NetworkFile = map[string]string{}
	}

	if v := c.NetdevFile["output"]; v == "" {
		c.NetdevFile["output"] = fmt.Sprintf("%s/%s.netdev", pathSN, c.IfName)
	}
	if v := c.NetdevFile["template"]; v == "" {
		c.NetdevFile["template"] = fmt.Sprintf("%s/%s.netdev.tmpl", pathSN, c.IfName)
	}

	if v := c.NetworkFile["output"]; v == "" {
		c.NetworkFile["output"] = fmt.Sprintf("%s/%s.network", pathSN, c.IfName)
	}
	if v := c.NetworkFile["template"]; v == "" {
		c.NetworkFile["template"] = fmt.Sprintf("%s/%s.network.tmpl", pathSN, c.IfName)
	}

	return nil
}

func writeFiles(cli *CLI, tun *pia.Tunnel) {
	if fs, err := fileops.Parse(cli.NetdevFile); err != nil {
		log.Panicf("Invalid --netdev-file: %v", err)
	} else if err := fs.Generate(tun); err != nil {
		log.Panicf("Could not generate netdev file: %v", err)
	}
	if fs, err := fileops.Parse(cli.NetworkFile); err != nil {
		log.Panicf("Invalid --network-file: %v", err)
	} else if err := fs.Generate(tun); err != nil {
		log.Panicf("Could not generate network file: %v", err)
	}
}

func main() {
	var cli CLI
	kong.Parse(&cli, kong.Name("pia-setup-tunnel"))

	// If directed to use cached info, just read the cache and write the files
	if cli.FromCache {
		// grab the cached tunnel info
		tun, err := pia.ReadCache(cli.CacheDir, cli.IfName)
		if err != nil {
			log.Panicf("Could not read cache: %v", err)
		}
		writeFiles(&cli, tun)
		return
	}

	// Find the "best" reg_id if requested
	var reg *pia.Region
	if cli.Region == "auto" || cli.Region == "" {
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
		reg, err = pia.FindRegion(cli.Region)
		if err != nil {
			log.Panicf("%v", err)
		}
	}

	// Create a Tunnel struct and populate it with fresh WG keys and an access token
	tun := pia.NewTunnel(reg, cli.IfName)
	defer func() {
		if err := tun.SaveCache(cli.CacheDir); err != nil {
			log.Panicf("Could not save cache: %v", err)
		}
	}()
	if err := genKeypair(tun, cli.WGBinary); err != nil {
		log.Panicf("Could not generate keypair: %v", err)
	}
	if !tun.Token.Valid() {
		if err := tun.NewToken(cli.Username, cli.Password); err != nil {
			log.Panicf("Could not get token: %v", err)
		}
	}

	// Register the WG keys to our account (identified by access token)
	if err := tun.Activate(); err != nil {
		log.Panicf("Could not register public key: %v", err)
	}

	// Finally, populate the templates
	writeFiles(&cli, tun)

	fmt.Println(tun.Status)
}

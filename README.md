# pia-tools

A suite to assist establishing a [wireguard][] tunnel to [PIA][], a VPN service
provider, under a [systemd-networkd][] Linux network stack using [PIA's new
REST API](https://github.com/pia-foss/manual-connections). Also includes
a utility for setting up, activating, and maintaining a forwarded port. To top
it off, the port forwarding utility can also notify [rtorrent][] of the port
using XML-RPC. [rtorrent][] can then advertise the forwarded port as it's
torrent port, and thereby receive incoming peer requests, provided you also set
up your firewall to also forward the port internally to the [rtorrent][] server
(you're on your own there, but what I do is simply forward almost all ports on
the firewall's wireguard interface to the rtorrent server).

According to unofficial documentation, [PIA][] requires you to refresh the port
forwarding assignment every few minutes. The `pia-portforward` utility can do
this with the `-refresh` flag.

## Caveat emptor

This code is only lightly tested with my own private set up. It may or may not
work. Patches welcome.

## Install

    go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
    go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest

## Configure

1. Set up `systemd.netdev` and `systemd.network` template files in
   `/etc/systemd/network` (e.g. `/etc/systemd/network/wg_pia.netdev.tmpl` and
   `/etc/systemd/network/wg_pia.network.tmpl`). These templates use the Go
   package [`text/template`](https://pkg.go.dev/text/template) to replace
   tokens with data received from [PIA][] when requesting the tunnel to be set
   up. For example:

    `/etc/systemd/network/<interface>.netdev.tmpl`
    ```
    [NetDev]
    Name={{ .Interface }}
    Kind=wireguard

    [WireGuard]
    PrivateKey={{ .PrivateKey }}

    [WireGuardPeer]
    PublicKey={{ .ServerPubkey }}
    AllowedIPs=0.0.0.0/0
    Endpoint={{ .ServerIp }}:{{ .ServerPort }}
    PersistentKeepalive=25
    ```

    `/etc/systemd/network/<interface>.network.tmpl`
    ```
    {{- $if := .Interface -}}
    {{- $gw := .ServerVip -}}
    [Match]
    Name={{ $if }}
    Type=wireguard

    [Network]
    Address={{ .PeerIp }}/32
    {{- range .DnsServers }}
    DNS={{ . }}%{{ $if }}
    {{- end }}

    [Route]
    Destination=0.0.0.0/0
    Gateway={{ $gw }}
    GatewayOnLink=yes

    [Route]
    Destination={{ $gw }}/32
    Scope=link

    {{ range .DnsServers -}}
    [Route]
    Destination={{ . }}/32
    Gateway={{ $gw }}

    {{ end -}}
    ```

    For info on what template fields are available, check out [the Tunnel
    struct in the `pia` package][tun].

2. `mkdir /var/cache/pia` and set the directory permissions as restrictive as
   you can, probably `root:root` and mode `0700`. This directory will hold
   `.json` files to cache information including your personal access tokens and
   wireguard private keys. These files do not store your [PIA][] username or
   password, but are still private!

3. Run `pia-setup-tunnel -username <user> -password <pass> -ifname wg_pia` as
   root to create the `.network` and `.netdev` files.

4. Run `systemctl restart systemd-networkd` as root to reload your network
   stack and activate the tunnel.

That's it, you have a VPN as your ipv4 default route. Use something like `curl
-4 icanhazip.com` to verify that the ip address is coming from PIA. Every few
days or weeks, repeat steps 3 and 4 to establish a new tunnel. They are good
for some period of time, but I replace mine once a week for privacy reasons.

If this tunnel is on a firewall, and if you also want port forwarding through
the firewall to an internal (natted) server, then, after confirming the tunnel
is up, do something like the following. (The following steps are *not
necessary* if you are simply setting up a tunnel on a client machine.)

1. Run `pia-portforward -ifname <interface> -rtorrent
   http://<rtorrent-ip>:<rtorrent-port>`. This will request a forwarding port,
   activate it, and then inform [rtorrent][] about the port. You may have to
   configure [rtorrent][] to accept XML-RPC queries on the `/RPC2` endpoint.

   **Note:** Don't include the `/RPC2` URL component in the `-rtorrent`
   parameter, as this is added automatically.

2. Make sure your firewall rules are forwarding the port. (You can inspect or
   parse `/var/cache/pia/<interface>.json` to determine the port number).

3. Every 15 minutes or so (presumably using `cron` or similar), run
   `pia-portforward -ifname <interface> -refresh` in order to refresh the
   assignment.

## TODO

- [ ] Make `systemd.service` and `systemd.timer`files for various phases of
  the tunnel lifecycle
- [ ] Test under more scenarios


[systemd-networkd]: https://www.freedesktop.org/software/systemd/man/systemd.network.html
[wireguard]: https://www.wireguard.com/
[PIA]: https://www.privateinternetaccess.com/
[rtorrent]: https://github.com/rakshasa/rtorrent
[tun]: https://github.com/jdelkins/pia-tools/blob/09ebfbe23d457cca3bf28a0a9a27c028311bc752/internal/pia/pia.go#L20

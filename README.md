# pia-tools

A suite to assist establishing a [wireguard][] tunnel to [PIA][], a VPN service
provider, under a [systemd-networkd][] Linux network stack using [PIA's new
REST API](https://github.com/pia-foss/manual-connections). Also includes
a utility for setting up, activating, and maintaining a forwarded port. To top
it off, the port forwarding utility can also notify [rtorrent][] and/or
[transmission][] of the port using XML-RPC. The torrent client(s) can then
advertise the forwarded port as it's torrent port, and thereby receive incoming
peer requests, provided you also set up your firewall to forward the port
internally (you're on your own there, but what I do is simply forward almost
all ports on the firewall's wireguard interface to the torrent server. It
would be almost as easy to update a firewall rule, if this router is also the
firewall).

According to unofficial documentation, [PIA][] requires you to refresh the port
forwarding assignment every few minutes. The `pia-portforward` utility can do
this with the `-refresh` flag.

## Caveat emptor

This code is only lightly tested with my own private set up. It may or may not
work. Patches welcome, especially if they would cover any more general use cases.

## About

[Private Internet Access][PIA] is, IMHO, a solid VPN provider.[^1] In their
latest generation infrastructure, they designed a system for use by their
mobile and desktop apps, which allow wireguard tunnels with randomly generated,
frequently rotated private keys. Unfortunately their Linux app, though
I haven't tried it, doesn't feel right for my use case, wherein I run the VPN
on my firewall, a Linux box running [systemd-networkd][], in the manner
described below. Luckily, however, [PIA][] published their REST API, which is
pretty simple to use; powered by that, we can dynamically configure our
systemd-networkd borne VPN tunnel and routing rules with this tool suite.

There are a couple of ways to use a VPN. One way is to set it up on your
workstation, laptop, or mobile device, so that your traffic is encrypted
through your lan and then the internet to the VPN service provider. For most
users, this is enough, and for that [PIA][] has nice desktop and mobile apps
that probably work fine.

Another way, if you control your network infrastructure, is to set up your
LAN's firewall to connect to the VPN service and route all (or some selection)
of the outgoing LAN traffic through the VPN. This way, any devices on the LAN
can benefit from the VPN without configuring those devices at all.[^2]

[PIA][] also has a dynamic port forwarding feature that allows you to run
a server on or behind your VPN endpoint.

Because of the dynamic nature of your enpoint public and virtual IP addresses,
Wireguard keys, and forwarded ipv4 port, it requires some tooling to set up.

[^1]: I have no affiliation with [PIA][] other than as a customer.

[^2]: One downside of this approach is that network traffic is still
  unencrypted on the LAN, so if you have any reason to fear privacy gaps at the
  physical LAN level despite your controlling the firewall, this approach is
  not recommended; stick with the official client app for true end-to-end
  encryption.

## Install Tools

    go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
    go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest     # optional
    go install github.com/jdelkins/pia-tools/cmd/pia-listregions@latest     # optional

## Configure Tunnel Interface

1. Set up `<interface>.netdev.tmpl` and `<interface>.network.tmpl` template
   files in `/etc/systemd/network/`. These templates use the Go package
   [`text/template`](https://pkg.go.dev/text/template) to replace tokens with
   data received from [PIA][] when requesting the tunnel to be set up. For
   example:

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

    The above examples should be pretty self-explanatory; if not, you should
    read up on `systemd-networkd` and/or `text/template`. For info on what
    other template fields are available (though the above examples demonstrate
    (I think) all of the useful ones), check out [the Tunnel struct in the
    `pia` package][tun]. The template processing package includes [sprig][],
    which provides a number of additional template functions, should they come
    in handy.

2. `mkdir /var/cache/pia` and set the directory permissions as restrictive as
   you can, probably `root:root` and mode `0700`. This directory will hold
   `.json` files to cache information including your personal access tokens and
   wireguard private keys. These files do not store your [PIA][] username or
   password, but should still be treated as private.

3. Run `pia-setup-tunnel --username <user> --password <pass> --ifname <interface>`
   as root to create the `.network` and `.netdev` files corresponding to the
   templates created above.

4. Inspect the generated files. If everything looks okay, run `systemctl
   restart systemd-networkd` as root to reload your network stack and activate
   the tunnel.

That's it, you should have a VPN as your ipv4 default route (asssuming you
configured the `.network` file as such). Use something like `curl -4
icanhazip.com` to verify that the ip address is coming from PIA. Every few days
or weeks, repeat steps 3 and 4 to establish a new tunnel. They are good for
some period of time, but I recommend replacing it once a week for privacy
reasons.

## Enabling Port Forwarding

If you also want incoming traffic on a single TCP port and a single UDP port
(both with the same port number) to be forwarded to your VPN endpoint, then
follow the procedure below.

**Note ☞**  The following steps are *not necessary* if you are simply setting
up a tunnel on a gateway. This tool is to enable running a server behind (or
on) the gateway that accepts incoming connections to be accessible through the
VPN.

1. Make sure your wireguard tunnel to PIA is up and running per the above
   procedure. The following step presumes that you have a working route to
   PIA's Wireguard endpoint virtual IP configured and working. It also assumes
   that, as part of running the above procedure, the `pia-setup-tunnel` tool
   saved some information in the file `/var/cache/pia/<interface>.json`. We
   will read that file and add to it as part of the next step.

2. If you are running [rtorrent][] or [transmission][] as the server behind the
   gateway, run, respectively, `pia-portforward -ifname <interface> -rtorrent
   http://<rtorrent-ip>:<rtorrent-port>` or `pia-portforward -ifname
   <interface> -transmission <transmission-ip>`. This will request a forwarding
   port from [PIA][], activate it, and then inform [rtorrent][] or
   [transmission][], respectively, about the port.

   - In the case of [rtorrent][], you may have to configure your instance to
     accept XML-RPC queries on the `/RPC2` endpoint. **Note ☞**  Don't include
     the `/RPC2` URL component in the `-rtorrent` parameter, as this is added
     automatically.

   - In the case of [transmission][], if your instance is configured with
     a username and password, provide those with the `-transmission-username`
     and `-transmission-password` parameters


   If you don't have a bittorrent server running (or just don't want to use the
   forwarded port for that), then just leave off the `-rtorrent` and
   `-transmission` options. You're on your own to parse
   `/var/cache/pia/<interface>.json` to obtain the assigned port and do
   something with it, such as setting up a DNAT firewall rule. You could, for
   example:

    ```sh
     nft add chain inet filter pia_portfoward '{type nat hook prerouting priority -100; policy accept}'
     nft flush chain inet filter pia_portforward
     nft add rule inet filter pia_portforward tcp dport $(jq .PFSig.port /var/cache/pia/pia.json) dnat ip to 192.168.0.80:80
     ```

   ...which would create a nftables rule to forward incoming traffic on your
   assigned port (retrieved from the cache file by the shell out to `jq`)
   to an internal webserver running on port 80. This assumes your VPN wireguard
   interface is named `pia` and your web server is running on `192.168.0.80`.
   This sets up the mechanism for forwarding connections, but to accually permit
   such connections, you would also then have to also add a static rule to accept
   them, typically somewhere in the `forward` chain in nftables. Again, you're
   on your own: the possibilies are vast once you can get the forwarded port number.

4. If you are running the VPN endpoint on a firewall, make sure your firewall
   rules are forwarding or redirecting the port where you want it. (You can
   inspect or parse `/var/cache/pia/<interface>.json` to determine the port
   number if you need it). Alternatively, you can just run [rtorrent][] (or
   whatever) on the firewall host, and configure the server to listen on PIA's
   assigned port.

5. Every 15 minutes or so (using `cron` or similar), run `pia-portforward
   -ifname <interface> -refresh` in order to refresh the port forwarding
   assignment. If not, PIA may reclaim the port for another customer.

## TODO

- [x] Make `systemd.service` and `systemd.timer`files for various phases of
  the tunnel lifecycle
- [x] Test under more scenarios
- [ ] Implement PIA's dynamic IP (DIP) feature


[systemd-networkd]: https://www.freedesktop.org/software/systemd/man/systemd.network.html
[wireguard]: https://www.wireguard.com/
[PIA]: https://www.privateinternetaccess.com/
[rtorrent]: https://github.com/rakshasa/rtorrent
[transmission]: https://transmissionbt.com/
[tun]: https://github.com/jdelkins/pia-tools/blob/09ebfbe23d457cca3bf28a0a9a27c028311bc752/internal/pia/pia.go#L20
[sprig]: http://masterminds.github.io/sprig/

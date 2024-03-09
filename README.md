# pia-tools

A suite to assist establishing a [wireguard][] tunnel to [PIA][], a VPN service
provider, under a [systemd-networkd][] Linux network stack using [PIA's new
REST API](https://github.com/pia-foss/manual-connections).

Also includes `pia-portforward`, a utility for setting up, activating, and
maintaining a forwarded port. This utility can also notify [rtorrent][] and/or
[transmission][] of the port number using their disinct RPC interfaces,
which will allow them to receive incoming peer requests.

According to their documentation, [PIA][] requires you to refresh the port
forwarding assignment every few minutes. The `pia-portforward` utility can do
this with the `--refresh` flag.

One final utility, `pia-listregions`, will show you, on your terminal, a
ranked (by ping time) list of PIA servers, with some possibly-salient info
about them, with the goal of helping you choose the 'best' region to connect to
if you are otherwise indifferent about their geography.

## Caveat emptor

This code is only lightly tested with my own private set up. It may or may not
work for you, and if it does, some assembly will be required by you. PRs are
welcome, especially if they would cover any more general use cases.

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
on your device, through your lan and then the internet to the VPN service provider.
This is "end to edge" encryption, and is the best protection a VPN can provide.
For most users, this is easy and affords plenty of protection, and for that
[PIA][] has nice desktop and mobile apps that are simple to use, but do require
installing them and setting them up. PIA are putting out apps for more and more
devices, but, unfortunately, they can't keep up with all the IoT devices, smart
home hubs, smart TV's, etc. As a result, we have to live with some unencrypted
traffic, and this segment is growing very fast in terms of number of devices
and the number of security issues arising from them.

Another way to use a VPN, available to you only if you control your network
infrastructure, is to set up your LAN's firewall to connect to the VPN service
and route all (or some selection) of the outgoing LAN traffic through the VPN.
This way, any devices on the LAN can benefit from the VPN without configuring
those devices at all. Note that this is "edge to edge" encryption, which is
a bit weaker since there is part of the path, the first part on your LAN, where
traffic is unencrypted. I'm okay with this, because I have physical control of
all apparatus from "end to edge". If you use any one else's wifi access point, or
connect your access point to someone else's switch, this would not be the case for
you. Stick with the VPN app. This project is for people who, like me, are
okay with this depth-for-breadth trade off, and who also use `systemd-networkd`
on their firewall.[^2]

[PIA][] also has a dynamic port forwarding feature that allows you to run
a server on or behind your VPN endpoint. This suite can help you with that.

Because of the dynamic nature of your enpoint public and virtual IP addresses,
Wireguard keys, and forwarded ipv4 port, it requires some tooling to set up.
Read on for how.

[^1]: I have no affiliation with [PIA][] other than as a customer.

[^2]: I suspect it would be trivial to adapt this tool suite to another 
wireguard-affine network stack, especially if it is configurable with text
files. Heck, the official wireguard client is written in go if I understand,
so you could probably, almost as easily, directly set up the tunnels with
library calls. I'm not likely to write that code though since I am happy with
`systemd-networkd` and I like having as much of my networking configuration
handled by it as possible. Check out the [Alternatives](#alternatives) though.

## Alternatives

| Tool | Description |
| - | - |
| [piawgcli](https://gitlab.com/ddb_db/piawgcli) | Similar approach, but generates a wireguard-cli configuration file instead of systemd-networkd |
| [Official](https://github.com/pia-foss/manual-connections) | Bash scripts that accomplish a similar goal using plain-jane networking commands, like `ip` `wg-quick` and so on. Also has a nice list of links to other alternatives. |

## Install Tools

    go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
    go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest     # optional
    go install github.com/jdelkins/pia-tools/cmd/pia-listregions@latest     # optional

## Configure Tunnel Interface

1. Set up `/etc/systemd/network/<interface>.netdev.tmpl` and `/etc/systemd/network/<interface>.network.tmpl` template
   files. These templates use the Go package
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

2. For bittorrent users: the thing with bittorrent, like ftp, is that some layer 3 info
   (the port number by which your server is reachable) is communicated in the layer 7
   protocol (announce messages). This layer transgression requires communicating some
   layer 3 info to your bittorrent server using some facility provided by the server.
   Most popular bittorrent servers that I've looked into have an RPC interface, which
   may need to be configured/enabled (see below).
   
   The `pia-portforward` tool can help communicating this port number to two
   popular bittorrent apps, namely [rtorrent][] and [transmission][]. (I haven't gotten
   arround to implementing a similar feature for [qbittorrent][], the single most
   popular bittorrent server on seedboxes, because I personally don't use it and
   no one has asked, but I believe the RPC capability is there and it should be be
   simple to add.)

   If you are running [rtorrent][] or [transmission][] as the server behind the
   gateway, run, respectively, `pia-portforward --ifname <interface> --rtorrent
   http://<rtorrent-ip>:<rtorrent-port>` or `pia-portforward --ifname
   <interface> --transmission <transmission-ip>`. This will request a forwarding
   port from [PIA][], activate it, and then inform [rtorrent][] or
   [transmission][], respectively, about the port.

   - In the case of [rtorrent][], you may have to configure your instance to
     accept XML-RPC queries on the `/RPC2` endpoint. **Note ☞**  Don't include
     the `/RPC2` URL component in the `--rtorrent` parameter, as this is added
     automatically.

   - In the case of [transmission][], if your instance is configured with
     a username and password, provide those with the `--transmission-username`
     and `--transmission-password` parameters

3. If you don't have a bittorrent server running (or just don't want to use the
   forwarded port for that), then just leave off the `--rtorrent` and
   `--transmission` options. You're on your own to parse
   `/var/cache/pia/<interface>.json` to obtain the assigned port and do
   something with it, such as setting up a DNAT firewall rule. Most protocols
   (http, for example) don't require the server to know the port number. (The
   server process has to listen to a configured port, but, with nat firewalls, this
   doesn't have to be the port that the client connects to). Such traffic can be handled
   cleanly on the firewall by dnatting traffic delivered to PIA's forwarded port to a
   static server port, locally on the firewall or elsewhere on the LAN side.
   You could, for example:

    ```sh
     nft add chain inet filter pia_portfoward '{type nat hook prerouting priority -100; policy accept}'
     nft flush chain inet filter pia_portforward
     nft add rule inet filter pia_portforward tcp dport $(jq .PFSig.port /var/cache/pia/pia.json) dnat ip to 192.168.0.80:80
     ```

   ...which would create a nftables rule to forward incoming traffic on your
   assigned port (retrieved from the cache file by the `$(jq .PFSig.port /var/cache/pia/pia.json)` part)
   to an internal webserver running on `192.168.0.80` at port 80. This example assumes your VPN wireguard
   interface is named `pia`.
   
   **Note ☞** The example above sets up the mechanism for forwarding connections.
   Assuming you don't run a default-accept firewall (hopefully you don't), then
   to accually permit such connections, you would also have to also add a rule to accept
   them, typically somewhere in the `forward` chain in nftables. This is not
   a lesson in firewall design, so again, you're on your own: the possibilies
   are numerous once you can get the forwarded port number.

4. Every 15 minutes or so (using systemd timers or similar), run `pia-portforward
   --ifname <interface> --refresh` in order to refresh the port forwarding
   assignment using your cached authentication token. If you don't do this,
   PIA will eventually reclaim the port for another customer, and traffic
   meant for you could be delivered somewhere else! You don't want that.
   My testing suggests that PIA is pretty conservative in this area, so
   I think most reasonable failure cases (e.g. your ISP goes down for a
   few hours, so you can't refresh) should be fine, but you definitely want
   to keep refreshing the assignment as long as you're intending to use it.


[systemd-networkd]: https://www.freedesktop.org/software/systemd/man/systemd.network.html
[wireguard]: https://www.wireguard.com/
[PIA]: https://www.privateinternetaccess.com/
[rtorrent]: https://github.com/rakshasa/rtorrent
[transmission]: https://transmissionbt.com/
[qbittorrent]: https://www.qbittorrent.org/
[tun]: https://github.com/jdelkins/pia-tools/blob/09ebfbe23d457cca3bf28a0a9a27c028311bc752/internal/pia/pia.go#L20
[sprig]: http://masterminds.github.io/sprig/

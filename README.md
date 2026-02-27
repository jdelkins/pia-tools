# pia-tools

A suite of tools for establishing a [WireGuard][wireguard] tunnel to [Private
Internet Access][PIA] on Linux using [systemd-networkd][], based on [PIA’s REST
API](https://github.com/pia-foss/manual-connections). It can also manage PIA’s
port forwarding feature and optionally notify rTorrent and/or Transmission (via
RPC) of the assigned port; this will allow receiving incoming BitTorrent peer
connections over the VPN.

A helper utility, `pia-listregions`, will show you, on your terminal, a ranked
(by ping time) list of PIA servers, with some possibly-salient info about them,
with the goal of helping you choose the 'best' region to connect to if you are
otherwise indifferent about their geography.

## Stability and Usability

This code is tested with only my own private setup. It may or may not work for
you, and if it does, some assembly will be required by you. That said, I have
used and developed it for a few years, and really enjoy the low-maintenance
functionality and peace of mind it provides for me.

These tools don't do anything exotic; they generate configuration
files and then standard system tools apply them. Nevertheless, it is for a
fairly advanced use case, not something for the average consumer.

If you identify additional features or encounter problems, I encourage you to
submit an issue or PR.

If you use this on a remote firewall, ensure you have out-of-band access before
experimenting with routing.

## Security Model

- The cache contains WireGuard private keys and PIA API auth material (e.g.,
  tokens/port-forward signatures). Treat it as sensitive.

- A new WireGuard keypair is generated each time the tunnel is (re-)established.

- The tools read PIA username/password (from env / envFile) but do not persist
  them to disk.

- API calls and cache updates can be performed as an unprivileged service
  account.

- Root privileges are required only to write into privileged locations
  (`/etc/systemd/network`) and to apply link changes.

- Intended to be automated via systemd; the included units invoke standard
  system tools (`networkctl`, `ip`) to activate changes.

## Quick Start

### NixOS

See: [NixOS Module](#nixos-module)

1. Import module
2. Write/configure envFile
3. Supply templates, or copy/modify examples into your flake
4. Build and deploy

### Other Distro

See [Manual Install](#manual-install) and the example units in [`./systemd`](./systemd)

1. Install tools
2. Write templates, or copy and modify the examples
3. Create a `pia` service account
4. Write environment file containing your credentials
5. Install and activate systemd service and timer
6. (Optional) Configure and activate port forwarding service and timer

## About

[Private Internet Access][PIA] is, in my opinion, a solid VPN provider.[^1]
In their latest generation infrastructure, they designed a system for use by
their mobile and desktop apps, which allow WireGuard tunnels with randomly
generated, frequently rotated private keys. Unfortunately, their Linux app,
though I haven't tried it, doesn't feel right for my use case, wherein I run
the VPN on my firewall, a Linux box running [systemd-networkd][], in the manner
described below. Luckily, however, [PIA][] published their REST API, which
is pretty simple to use; powered by that, we can dynamically configure our
systemd-networkd managed VPN tunnel and routing rules with this tool suite.

[PIA][]'s VPN service includes a dynamic port forwarding feature that allows
you to run a server on or behind your VPN endpoint. This suite can help you
with that.

Because of the dynamic nature of your endpoint public and virtual IP addresses,
WireGuard keys, and forwarded IPv4 port, it requires some tooling to set up.
Read on for how.

### Why run VPN on a gateway?

There are a couple of ways to use a VPN. One way is to set it up on your
workstation, laptop, or mobile device, so that your traffic is encrypted on
your device, through your LAN and then the internet to the VPN service provider.
This is "end-to-end" encryption, and is the best protection a VPN can provide.
For most users, this is easy and affords plenty of protection, and for that
[PIA][] has nice desktop and mobile apps that are simple to use, but do require
installing them and setting them up. PIA is putting out apps for more and more
devices, but, unfortunately, they can't keep up with all the IoT devices, smart
home hubs, smart TVs, etc. As a result, we have to live with some unencrypted
traffic, and this segment is growing very fast in terms of number of devices and
the number of security issues arising from them.

Another way to use a VPN, available to you only if you control your network
infrastructure, is to set up your LAN's firewall to connect to the VPN service
and route all (or some selection) of the outgoing LAN traffic through the VPN.
This way, any devices on the LAN can benefit from the VPN without configuring
those devices at all. Note that this is "edge-to-edge" encryption, which is a
bit weaker since there is part of the path, the first part on your LAN, where
traffic is unencrypted. I'm okay with this, because I have physical control of
all apparatus on my LAN. If you use anyone else's wifi access point, or connect
your access point to someone else's switch, this would not be the case for you.
Stick with the VPN app. This project is for people who, like me, are okay with
this depth-for-breadth tradeoff, and who also use `systemd-networkd` on their
firewall.[^2]

Because you can, _should_ you do this? I find little downside in using a fast
[WireGuard][wireguard]-based VPN. My VPN tunneled traffic is indistinguishably
fast as untunneled traffic almost all of the time, and this suite is pretty
much hands-free for me. So is it worth the tradeoff? For me, yes, only because
the tradeoff is close to zero. I wouldn't do it if I had to make a meaningful
performance sacrifice or take on a significant administration burden.

[^1]: I have no affiliation with [PIA][] other than as a customer. My opinions
are my own, and I cannot endorse their, or any, VPN service. Caveat emptor.

[^2]: I suspect it would be trivial to adapt this tool suite to another 
WireGuard-friendly network stack, especially if it is configurable with text
files. Heck, the official WireGuard client is written in go if I understand,
so you could probably, almost as easily, directly set up the tunnels with
library calls. I'm not likely to write that code though since I am happy with
`systemd-networkd` and I like having as much of my networking configuration
handled by it as possible. Check out the [Alternatives](#alternatives) though
if you have a different networking preference.

## Alternatives

| Tool                                                       | Description                                                                                                                                                             |
|------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [piawgcli](https://gitlab.com/ddb_db/piawgcli)             | Similar approach, also written in go, but without the port forwarding and NixOS features, and generates a wireguard-cli configuration file instead of systemd-networkd. |
| [Official](https://github.com/pia-foss/manual-connections) | Bash scripts that accomplish a similar goal using plain-jane networking commands, like `ip`, `wg-quick` and so on. Also has a nice list of links to other alternatives. |

## NixOS Module

This repo includes a NixOS module to make it easy to configure and deploy
on that OS. Just include it as an input in your flake, and configure
through the `services.pia-tools` option tree. You would probably
also want to include (your modified version of) the systemd-networkd
template files ([`pia.netdev.tmpl`](./systemd/network/pia.netdev.tmpl)
and [`pia.network.tmpl`](./systemd/network/pia.network.tmpl)) in your
flake repo, or write your own, or base your config on the [detailed
example](#nixos-detailed-example-configuration), which follows.

### NixOS Module options

| Option                                   | Type                                | Description                                                                                                                                                                                                                                  |
|------------------------------------------|-------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `services.pia-tools.enable`              | `bool`                              | Enable the pia-tools NixOS module.                                                                                                                                                                                                           |
| `services.pia-tools.package`             | `package`                           | The pia-tools package to use.                                                                                                                                                                                                                |
| `services.pia-tools.user`                | `string`                            | User to run the tool as.                                                                                                                                                                                                                     |
| `services.pia-tools.group`               | `string`                            | Group to run the tool as.                                                                                                                                                                                                                    |
| `services.pia-tools.cacheDir`            | `path`                              | Where to store tunnel descriptions in JSON format, containing private keys.                                                                                                                                                                  |
| `services.pia-tools.ifname`              | `string`                            | Name of PIA WireGuard network interface.                                                                                                                                                                                                     |
| `services.pia-tools.region`              | `string`                            | Region to connect to, or `auto` by default.                                                                                                                                                                                                  |
| `services.pia-tools.rTorrentUrl`         | `null or string`                    | URL to rTorrent XML-RPC endpoint.                                                                                                                                                                                                            |
| `services.pia-tools.transmissionUrl`     | `null or string`                    | Transmission RPC endpoint URL. If your Transmission server requires a username and password, set them in `config.services.pia-tools.envFile` with `TRANSMISSION_USERNAME` and `TRANSMISSION_PASSWORD`.                                       |
| `services.pia-tools.envFile`             | `path`                              | **Required.** Path to a file that sets environment variables used to set up the tunnel device. Recognized variables include `PIA_USERNAME` and `PIA_PASSWORD` (required), plus optional `TRANSMISSION_USERNAME` and `TRANSMISSION_PASSWORD`. |
| `services.pia-tools.resetServiceName`    | `string`                            | Name of systemd service for pia-tools tunnel reset.                                                                                                                                                                                          |
| `services.pia-tools.resetTimerConfig`    | `null or systemd timerConfig attrs` | Timer defining frequency of resetting the tunnel. Set to `null` to disable.                                                                                                                                                                  |
| `services.pia-tools.refreshServiceName`  | `string`                            | Name of systemd service for pia-tools tunnel port forwarding refresh (only relevant if portForwarding is enabled).                                                                                                                           |
| `services.pia-tools.refreshTimerConfig`  | `null or systemd timerConfig attrs` | Timer defining frequency of refreshing the tunnel's port forwarding assignment. Set to `null` to disable.                                                                                                                                    |
| `services.pia-tools.whitelistScript`     | `null or path`                      | Script to run when the WireGuard endpoint is established (e.g., add the endpoint IP to a firewall passlist). The script is called with the IP as the only argument. Set to `null` to ignore.                                                 |
| `services.pia-tools.portForwarding`      | `bool`                              | Whether to request a port forwarding assignment from PIA.                                                                                                                                                                                    |
| `services.pia-tools.netdevTemplateFile`  | `path`                              | `systemd.netdev` template used to generate the actual `.netdev`.                                                                                                                                                                             |
| `services.pia-tools.networkTemplateFile` | `path`                              | `systemd.network` template used to generate the actual `.network`.                                                                                                                                                                           |
| `services.pia-tools.netdevFile`          | `path`                              | Path at which to install the generated `.netdev` file.                                                                                                                                                                                       |
| `services.pia-tools.networkFile`         | `path`                              | Path at which to install the generated `.network` file.                                                                                                                                                                                      |

### NixOS Detailed Example Configuration

This example is not likely to be usable by you directly. These are complex
networking topics, and it is dangerous to rely on examples for your production
set-up. Do your homework and make sure you have a correct and safe firewall
configuration.

In this example, we are configuring a firewall host, with a single LAN
interface:

- pia-tools gets configured to establish a VPN tunnel called `wg_pia`, using
  inline-defined systemd-networkd templates.
- The region we connect to will be automatically chosen each time the tunnel is
  reestablished. It will choose the lowest ping-time server that supports port
  forwarding.
- These templates set up a default route through the tunnel in a dedicated
  routing table, which is used only by traffic from the LAN. The included
  routing-policy routes forwarded LAN traffic through the VPN while keeping the
  firewall host’s own traffic (including tunnel establishment) on the WAN.
- We have a transmission server running on the LAN at 192.168.100.100, and we
  want to forward UDP ports to it from the VPN.
- All outgoing IPv4 traffic from the LAN is allowed out through the VPN, from
  transmission or any other host on the LAN that uses this firewall as its
  router.
- In the example, if the VPN link goes down systemd-networkd will remove the VPN
  default route, in which case LAN traffic should fall back to the WAN default
  route, unless you enable the optional kill switch.
- The tunnel gets torn down and remade with a new virtual IP every day at 5:15
  AM.

#### Notes

- Theoretically, there is no reason this gateway has to be exposed on your WAN,
  it could, for example, be in a DMZ behind your WAN router, with only certain
  LAN clients configured (manually or via DHCP) to use its LAN address as its
  default route. It just needs outbound WAN connectivity in order to establish
  the tunnel.

- To make this gateway the default for your LAN, you would probably want to
  configure your DHCPv4 server (if you use one, and you probably should) to
  advertise the router's LAN address as the default route (DHCP option 3).

- To make this example actually usable in any way, you would need to, at a
  minimum, configure the LAN and WAN interfaces. The example doesn't address
  that.

- As written, the example uses the nftables-backed firewall; extraForwardRules
  are written in nft syntax. If you prefer iptables (I don't), adapt accordingly.

#### Regarding DNS "leaks"

DNS leaks happen when you allow some traffic to bypass the tunnel, such as can
happen with IPv6 and DNS.

- **IPv6**. PIA doesn't provide IPv6 tunneling; it is an IPv4-only service. If
  you run an IPv4/IPv6 dual stack on the LAN network, any IPv6 traffic that
  exits via the WAN interface will bypass the VPN tunnel. Therefore, you may
  wish to add firewall and/or routing rules to block outgoing IPv6. As it is,
  this example doesn't enable, disable, or otherwise address IPv6 networking.

- **DNS**. Most connections start with a DNS lookup of a domain name. If
  that lookup is sent to a public DNS server via a route outside of the VPN
  tunnel, then you are leaking information about where you make connections.
  Even when DNS queries are sent through the VPN tunnel, the DNS provider
  can still log the lookup and its timing; your ISP, however, only sees
  encrypted traffic to the VPN endpoint and cannot observe the DNS contents.
  Best practice is to use the VPN-provided DNS servers, which are ostensibly
  zero-log. If you use DHCP on the LAN, you should therefore advertise PIA's
  DNS servers in option 6 (`domain-name-servers`). In practice, as of this
  writing, they are consistently `10.0.0.242` and `10.0.0.243`, but PIA could
  change them. The addresses are actually retrieved and cached in the json file
  by `pia-setup-tunnel`, and you could make use of this (in order to, e.g.,
  dynamically update your DHCP server configuration; I personally don't bother
  to do so) as the following snippet demonstrates. The example below adds routes
  to the (dynamically-retrieved) DNS servers, so they should work as long as
  your LAN subnet does not overlap with `10.0.0.242/31` (for example, if you are
  using a broad `10.0.0.0/8` LAN).

  ```
  $ jq -r '.dns_servers[]' /var/cache/pia/wg_pia.json
  10.0.0.243
  10.0.0.242
  ```

#### The NixOS Example

`flake.nix`
```nix
inputs = {
  nixpkgs.url = "...";
  pia-tools.url = "github:jdelkins/pia-tools";
};

outputs =
  { nixpkgs, pia-tools }:
  {
    nixosConfigurations.<host> = nixpkgs.lib.nixosSystem {
      system = "...";
      modules = [
        pia-tools.nixosModules.pia-tools
        (
          { pkgs, lib, ... }:
          let
            wgIf = "wg_pia";
            lanIf = "lan";
            transmissionIp = "192.168.100.100";
            # set to true to disable outbound networking when the VPN link is down
            killSwitch = false;

            netdevTemplate = pkgs.writeText "${wgIf}.netdev.tmpl" ''
              [NetDev]
              Name={{ .Interface }}
              Kind=wireguard

              [WireGuard]
              PrivateKey={{ .PrivateKey }}

              [WireGuardPeer]
              Endpoint={{ .ServerIp }}:{{ .ServerPort }}
              PublicKey={{ .ServerPubkey }}
              AllowedIPs=0.0.0.0/0
              PersistentKeepalive=25
            '';

            networkTemplate = pkgs.writeText "${wgIf}.network.tmpl" ''
              [Match]
              Name={{ .Interface }}

              [Network]
              Address={{ .PeerIp }}/32
              {{- range .DnsServers }}
              DNS={{ . }}
              {{- end }}

              [Route]
              Destination={{ .ServerVip }}/32
              Scope=link

              {{ range .DnsServers -}}
              [Route]
              Destination={{ . }}/32
              Gateway={{ $.ServerVip }}
              GatewayOnLink=true

              {{ end -}}

              # NOTE: Default route goes in separate routing table. RPDB rules
              #       must direct traffic to this table to use it.
              [Route]
              Destination=0.0.0.0/0
              Gateway={{ .ServerVip }}
              GatewayOnLink=true
              Scope=global
              Table=vpn
            '';
          in
          {
            # Create a routing table mapped to the name "vpn", which will house
            # the default route for packets forwarded from the lan interface.
            # Local traffic originating on the firewall (importantly including
            # the encapsulating WireGuard link to PIA's server) will use the
            # regular default route in the "main" table.
            systemd.network.config = {
              routeTables.vpn = 1000;
              addRouteTablesToIPRoute2 = true;
            };

            systemd.network.networks.lo = {
              enable = true;
              name = "lo";
              # optional kill switch implementation
              routes = lib.optional killSwitch {
                Destination = "0.0.0.0/0";
                # the high metric means this route will only be used if the VPN
                # is down, in which case networkd will remove its route, leaving
                # only this.
                # Without this, if the VPN is down, the vpn table should be
                # empty, meaning the rules will fall thorough to the main
                # routing table, where the host's default route should match.
                Metric = 100000;
                Type = "unreachable";
                Table = "vpn";
              };
              # We attach RPDB rules to lo because it’s always present; the
              # rules apply globally.
              routingPolicyRules = [
                # First traverse the main table, but exclude default routes. If
                # you do LAN routing to multiple internal subnets, they should
                # be picked up cleanly in this way.
                {
                  Priority = 900;
                  SuppressPrefixLength = 0;
                  Family = "ipv4";
                  Table = "main";
                }
                # For LAN traffic that doesn't match site local routes above,
                # send them to the "vpn" table, which should have the VPN
                # default route, and possibly the "unreachable" kill switch, if
                # enabled. Local-origin traffic will skip this and go to main
                # via the default FIB rules.
                {
                  Priority = 1000;
                  IncomingInterface = lanIf;
                  Family = "ipv4";
                  Table = "vpn";
                }
              ];
            };

            services.pia-tools = {
              enable = true;
              ifname = wgIf;

              # Change this if you want to always connect to a certain regional
              # server.
              region = "auto";

              # envFile needs to contain
              #   PIA_USERNAME=<pia username>
              #   PIA_PASSWORD=<pia password>
              # and, if needed:
              #   TRANSMISSION_USERNAME=<transmission username>
              #   TRANSMISSION_PASSWORD=<transmission password>
              # You can put this file in place by hand, or use something like
              # sops-nix to generate it based on encrypted secrets stored in the
              # flake repo.
              envFile = "/etc/pia-secrets.sh";

              # If true, request and maintain a port forwarding
              # assignment after successfully establishing a tunnel.
              portForwarding = true;

              # RPC endpoint of transmission server. Usually the web
              # endpoint with /rpc added at the end.
              # Ignored unless portForwarding = true
              transmissionUrl = "http://${transmissionIp}:9091/rpc";

              # rotate the tunnel daily at 5:15 am
              resetTimerConfig.OnCalendar = "*-*-* 05:15:00";

              # Your templates. These could be maintained in the flake repo, or
              # externally. Here, we are writing the template files to the nix
              # store, and the path resolves the store path.
              netdevTemplateFile = netdevTemplate;
              networkTemplateFile = networkTemplate;
            };

            # minimal firewall configuration. Start with the NixOS firewall module
            boot.kernel.sysctl."net.ipv4.ip_forward" = 1;
            networking.firewall.enable = true;
            networking.nftables.enable = true;

            # masquerading and DNAT. masquerade LAN traffic out of the vpn, and forward
            # ports >10000 arriving from the vpn to our transmission server
            networking.nat = {
              enable = true;
              externalInterface = wgIf;
              internalInterfaces = [ lanIf ];
              forwardPorts = [
                {
                  proto = "udp";
                  sourcePort = "10000:65535";
                  destination = "${transmissionIp}:10000-65535";
                }
              ];
            };

            # Allow LAN clients (incl. Transmission host) to egress out the VPN.
            # Allow UDP traffic from the VPN link to transmission. PIA will also
            # forward TCP but we can just ignore it for bittorrent purposes.
            # networking.nat only sets up the NAT rules, we have to explicitly
            # allow the traffic.
            networking.firewall.extraForwardRules = ''
              iifname "${lanIf}" oifname "${wgIf}" accept
              iifname "${wgIf}" oifname "${lanIf}" ct state { established, related } accept
              iifname "${wgIf}" oifname "${lanIf}" ip daddr ${transmissionIp} udp dport 10000-65535 accept
            '';
          }
        )
        ...
      ];
      ...
    };
  };
```

## Manual Install

### Tunnel Establishment

1. Install the tools. This will require a go toolchain to be installed on your system.
   The systemd units expect the binary will be installed in `/usr/local/bin`. If you
   install somewhere else, you'll have to edit the `pia-reset-tunnel@.service` and
   `pia-pf-refresh@.service` units after you install them in step 5 below.

       sudo env GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
       sudo env GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-listregions@latest   # optional

2. The next steps assume you want the interface named `pia` (if not, replace
   path components with your preferred interface name).

   Set up `/etc/systemd/network/pia.netdev.tmpl` and
   `/etc/systemd/network/pia.network.tmpl` template files. These
   templates use the Go package [`text/template`][text-template] to replace
   tokens with data received from [PIA][] when requesting the tunnel to be set
   up. You can use the examples in [./systemd/network](./systemd/network)
   and/or modify them to suit you.

   These examples should be pretty self-explanatory; if not, you should read
   up on [systemd-networkd][] and/or [text/template][text-template]. For info
   on what other template fields are available (though the examples demonstrate
   (I think) all of the useful ones), check out [the Tunnel struct in the `pia`
   package](./internal/pia/pia.go#L18). The template processing package includes
   [sprig][], which provides a number of additional template functions, should
   they come in handy.

       sudo install -o root -g root -m 0644 ./systemd/network/pia.net*.tmpl /etc/systemd/network/
       sudoedit /etc/systemd/network/pia.netdev.tmpl
       sudoedit /etc/systemd/network/pia.network.tmpl

3. Create a service account under which to run the service, and a cache directory.

       sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin pia
       sudo install -d -o pia -g pia -m 0750 /var/cache/pia

   The `/var/cache/pia` directory will hold `.json` files to cache information
   including your personal access tokens and WireGuard private keys. These files
   do not store your [PIA][] username or password, but should still be treated
   as private.

4. Create/edit the environment file containing your PIA credentials. You can
   copy and edit [the example file](./systemd/pia.conf). Make it readable by the
   `pia` user.

       sudo install -o root -g pia -m 0640 ./systemd/pia.conf /etc/pia.conf
       sudoedit /etc/pia.conf

5. Install [the systemd service file](./systemd/system/pia-reset-tunnel@.service)
   and [the systemd timer file](./systemd/system/pia-reset-tunnel@.timer) into
   `/etc/systemd/system`, and then enable the tunnel reset timer.

       sudo install -o root -g root -m 0644 ./systemd/system/pia-*@.{timer,service} /etc/systemd/system/
       # optional, edit for binary paths:
       sudoedit /etc/systemd/system/pia-reset-tunnel@.service
       sudoedit /etc/systemd/system/pia-pf-refresh@.service

   When you're happy with the service and timer units, activate them.

       sudo systemctl daemon-reload
       sudo systemctl enable --now pia-reset-tunnel@pia.timer

   This will reset the tunnel once a week, on Wednesdays at 03:00 by default.

6. You can activate the tunnel now.

       sudo systemctl start pia-reset-tunnel@pia.service

   You can repeat this command whenever you want to forcibly tear down and
   rebuild the tunnel with a new ip (and forwarded port, if configured).
   It is a "oneshot" service, which, on invocation follows this basic
   procedure:

    - generates a new WireGuard keypair
    - if region is "auto", determine the closest port-forward capable region
    - registers the new WireGuard public key with PIA
    - gets the connection details from PIA based on the configured (or auto-selected) region
    - regenerates the systemd-networkd config files using the templates
    - tears down the existing tunnel interface, if it exists
    - tells networkd to build the interface from the new config

7. Ensure it's working.

       sudo networkctl status pia
       sudo wg show pia
       curl -4 ifconfig.me

8. Adjust your network routing, if you wish, to send traffic selectively out
   of the tunnel. You're on your own here, but for some clues, you might
   check out the [NixOS example](#the-nixos-example).

That's it, you should have a VPN as your IPv4 default route (assuming you
configured the `.network` file as such). Use something like `curl -4
icanhazip.com` to verify that the ip address is coming from PIA.

### Enabling Port Forwarding

If you also want incoming traffic on a single TCP port and a single UDP port
(both with the same port number) to be forwarded to your VPN endpoint, then
follow the procedure below.

**Note ☞**  The following steps are *not necessary* if you are simply setting
up a tunnel on a gateway. This tool is to enable running a server behind (or
on) the gateway that accepts incoming connections to be accessible through the
VPN.

1. Install the `pia-portforward` tool.

       sudo env GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest

2. If you run a bittorrent server on your LAN (or on the router itself),
   you can edit `/etc/pia.conf` to provide some additional details.

       sudoedit /etc/pia.conf

   For [transmission][], uncomment and set the variables

   - `TRANSMISSION`: set to the server's web interface URL, with `/rpc` at the end, e.g. `http://192.168.100.100:9091/rpc`
   - `TRANSMISSION_USERNAME`: if web access control is configured on transmission server, set to the username
   - `TRANSMISSION_PASSWORD`: if web access control is configured on transmission server, set to the password

   For [rtorrent][], you will need a reverse proxy (lighttpd,
   nginx, etc.) to front the SCGI interface via XMLRPC. See
   [here](https://github.com/rakshasa/rtorrent-doc/blob/master/RPC-Setup-XMLRPC.md)
   for instructions. These instructions indicate configuring the standard
   location `/RPC2` for the XMLRPC interface. Please follow this convention.
   Once that is up and running, you should set the `RTORRENT` variable to the
   url of this reverse proxy server. Leave off the `/RPC2` part of the URL;
   the tool adds this automatically.

   - `RTORRENT`: set to reverse proxy URL, after configuring the `/RPC2` endpoint. e.g. `http://192.168.100.101:5000`

   Other bittorrent servers are not supported currently, sorry.

3. If you don't have a bittorrent server running (or just don't want
   to use the forwarded port for that), you're on your own to parse
   `/var/cache/pia/pia.json` to obtain the assigned port and do something with
   it. For example, to forward the port to an internal web server, you could:

   ```sh
   IF=pia
   PORT="$(jq -r '.PFSig.port' "/var/cache/pia/${IF}.json")"
   WEBSERVER="192.168.0.80:80"

   nft add table ip nat 2>/dev/null || true
   nft add chain ip nat pia_portforward '{ type nat hook prerouting priority -100 }' 2>/dev/null || true
   nft flush chain ip nat pia_portforward
   nft add rule ip nat pia_portforward tcp dport $PORT dnat to $WEBSERVER
   ```

4. Enable the timer to refresh the port forwarding assignment every 15 minutes

       sudo systemctl enable --now pia-pf-refresh@pia.timer

5. To enable the port forwarding now, just re-build the tunnel.

       sudo systemctl start pia-reset-tunnel@pia.service

## CLI Usage

Although it is recommended to run these tools with the help of systemd timers,
the tools are perfectly valid to use via their CLI interface. The pia-tools
repository provides two primary command-line utilities:

- `pia-setup-tunnel`
- `pia-portforward`

These are designed to work together for configuring and maintaining a PIA
WireGuard tunnel and optional port forwarding.

Additionally, `pia-listregions`, which accepts no flags or other configuration,
simply downloads and lists the available regions as discussed above.

### pia-setup-tunnel

#### Description

`pia-setup-tunnel` creates or updates a WireGuard tunnel using the PIA API, writes
the corresponding systemd-networkd .netdev and .network files. The systemd
unit would invoke the utility, and then set up the interface for use. It is
typically, therefore, invoked from a systemd unit but may also be run manually.

The command:

- Authenticates to PIA using your username and password
- Selects a region
- Generates WireGuard keys and retrieves endpoint configuration
- Writes systemd-networkd configuration files

#### Required Inputs

Only the credential parameters are essential; defaults are provided for
everything else. Credentials may be provided via flags or environment variables:

| Parameter    | Environment Variable | Meaning              |
|--------------|----------------------|----------------------|
| `--username` | PIA_USERNAME         | PIA account username |
| `--password` | PIA_PASSWORD         | PIA account password |

#### Flags

| Flag                         | Environment Var | Default          | Meaning                                                                                                             |
|------------------------------|-----------------|------------------|---------------------------------------------------------------------------------------------------------------------|
| `--region string`            | PIA_REGION      | `auto`           | PIA region identifier (e.g., us_chicago, us_texas)                                                                  |
| `--username string`          | PIA_USERNAME    | _required_       | PIA account username                                                                                                |
| `--password string`          | PIA_PASSWORD    | _required_       | PIA account password                                                                                                |
| `--if-name string`           | _n/a_           | `pia`            | Interface name to create or reconfigure (e.g., v4, wg0)                                                             |
| `--netdev-file key=value,…`  | _n/a_           | _see below_      | Write a .netdev file using a key/value specification                                                                |
| `--network-file key=value,…` | _n/a_           | _see below_      | Write a .network file using a key/value specification                                                               |
| `--cache-dir`                | _n/a_           | `/var/cache/pia` | directory in which to save a json file with the tunnel parameters.                                                  |
| `--wg-binary`                | _n/a_           | `wg`             | path to the `wg` binary from wireguard-tools (look in $PATH by default)                                             |
| `--from-cache`               | _n/a_           | _unset_          | Skip accessing PIA's api, and just (re-)generate the networkd files from the json cache. Useful to debug templates. |

#### File Specification Format

Both `--netdev-file` and `--network-file` accept a comma-separated list
of key=value pairs. All of the keys are optional and will use defaults as
indicated.

| Key         | Default                                                         | Meaning                                    |
|-------------|-----------------------------------------------------------------|--------------------------------------------|
| `output=`   | `/etc/systemd/network/<ifname>.{network,netdev}`                | Path at which to save the generated file.  |
| `template=` | `/etc/systemd/network/<ifname>.{network,netdev}.tmpl`           | Path of the source template for this file. |
| `owner=`    | _invoking user_                                                 | The owner account name for the file.       |
| `group=`    | _invoking group_                                                | The group name for the file.               |
| `mode=`     | _runtime default from the environment (usually: 0666 & ~UMASK)_ | The file mode in octal (e.g. 0440)         |

Example:

`--netdev-file=output=/etc/systemd/network/pia.netdev,template=/etc/systemd/network/pia.netdev.tmpl,mode=0440,group=systemd-network`

#### Example Usage

__Minimal example using environment variables.__ This will generate
`/etc/systemd/network/pia.{netdev,network}` files using the templates
`/etc/systemd/network/pia.{netdev,network}.tmpl`. If the template files are not
found, you'll get an error. If run as non-root, the command will also fail, as
root is required to write the files to this system directory and to change the
group of the netdev file, as specified in this example. Note that, generally,
systemd-networkd will require read permission on netdev files and will fail if
the files are world-readable, for security reasons.

```sh
export PIA_USERNAME=youruser
export PIA_PASSWORD=yourpass
export PIA_REGION=us_chicago

pia-setup-tunnel --netdev-file=group=systemd-network,mode=0440
```

__Full explicit example.__ This will do the same as the above, without relying (as
much) on the defaults.

```sh
pia-setup-tunnel \
  --username youruser \
  --password yourpass \
  --region us_chicago \
  --if-name pia \
  --netdev-file=output=/etc/systemd/network/pia.netdev,template=/etc/systemd/network/pia.netdev.tmpl,owner=root,group=systemd-network,mode=0440 \
  --network-file=output=/etc/systemd/network/pia.network,template=/etc/systemd/network/pia.network.tmpl,mode=0444
```

#### Activating the tunnel

The example systemd service takes some extra steps after running `pia-setup-tunnel`
to activate the tunnel link. If you are running it on the cli, you may wish
to follow up with these steps, otherwise the tunnel will have been reconfigured,
but the changes not activated.

```sh
ip link set down dev '<ifname>' || true
ip link del '<ifname>' || true
networkctl reload
networkctl reconfigure '<ifname>'
networkctl up '<ifname>'
```

### pia-portforward

#### Description

`pia-portforward` requests and maintains a forwarded port from the active
PIA tunnel. It retrieves the assigned port and optionally updates downstream
services such as rTorrent or Transmission.

This command must be executed after the tunnel is active.

#### Flags

| Flag                             | Environment Var       | Default | Meaning                                                                 |
|----------------------------------|-----------------------|---------|-------------------------------------------------------------------------|
| `--username string`              | PIA_USERNAME          | _none_  | Used to get a new authentication token, if expired.                     |
| `--password string`              | PIA_PASSWORD          | _none_  | ibid                                                                    |
| `--if-name string`               | _n/a_                 | `pia`   | Interface name associated with the active PIA tunnel                    |
| `--rtorrent string`              | RTORRENT              | _none_  | rTorrent XML-RPC endpoint (e.g., http://localhost:5000)                 |
| `--transmission string`          | TRANSMISSION          | _none_  | Transmission RPC endpoint (e.g., http://localhost:9091/rpc)             |
| `--transmission-username string` | TRANSMISSION_USERNAME | _none_  | Transmission RPC username (if required)                                 |
| `--transmission-password string` | TRANSMISSION_PASSWORD | _none_  | Transmission RPC password (if required)                                 |
| `--refresh`                      | _n/a_                 | _unset_ | Don't get a new port forwarding assignment, just refresh the active one |

#### Example Usage

__Basic port forward retrieval.__ Will obtain a forwarding port, store it in the
cache file, but do nothing further with it.

```sh
pia-portforward --if-name pia
```

__Update rTorrent with the forwarded port.__ This will obtain a forwarding port
and communicate it to an rtorrent instance running locally on the router. If a
forwarding port is already active, this will grab a new (and likely different)
port.

```sh
pia-portforward \
  --if-name pia \
  --rtorrent http://127.0.0.1:5000
```

__Update Transmission with authentication.__ This will obtain a forwarding port
and communicate it to a transmission instance running at ip 192.168.77.77. This
example uses environment variables for the transmission related parameters, but
you could also use CLI flags. Uses the cached PIA token: if you just set up the
tunnel successfully, you shouldn't need the PIA username/password; the cached
token should still be valid. You can provide the PIA credentials if you want to
be sure.

```sh
export TRANSMISSION=http://192.168.77.77:9091/rpc
export TRANSMISSION_USERNAME=user
export TRANSMISSION_PASSWORD=pass

pia-portforward --if-name pia
```

__Refresh Port Forwarding Assignment.__ For reasons of practicality, this should
be called from a systemd timer or cron, but works fine from the CLI too, if you
can remember to do so every 15 minutes. Supply credentials, as the cached token
has a finite valid lifetime; with the username and password, we can grab a new
one if necessary.

```sh
export PIA_USERNAME=user
export PIA_PASSWORD=pass

pia-portforward --if-name pia --refresh
```

#### NixOS: Running CLI without installing

The project's flake includes "app" outputs for the three CLI programs, allowing
NixOS users to run the CLI programs from the github repo without installing
them. Examples:

```sh
# This will list the available PIA regions and their ping time
# using pia-listregions
nix run github:jdelkins/pia-tools

# This does the same thing, explicitly naming the app (listregions
# is the default app)
nix run github:jdelkins/pia-tools#listregions

# Runs pia-setup-tunnel --help
nix run github:jdelkins/pia-tools#setup-tunnel -- --help

# Runs pia-portforward --help
nix run github:jdelkins/pia-tools#portforward -- --help
```

Alternatively, you could use `nix shell` to make the CLI programs temporarily
available to you in a subshell.

```sh
nix shell github:jdelkins/pia-tools
pia-listregions
pia-setup-tunnel --help
pia-portforward --help
```

## Troubleshooting

The `pia-listregions` tool attempts to send a non-root ping. If this is not
permitted by default on your system, run the binary as root or else it must be
enabled with a sysctl command like the following:

    sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"


[systemd-networkd]: https://www.freedesktop.org/software/systemd/man/systemd.network.html
[wireguard]: https://www.wireguard.com/
[PIA]: https://www.privateinternetaccess.com/
[rtorrent]: https://github.com/rakshasa/rtorrent
[transmission]: https://transmissionbt.com/
[qbittorrent]: https://www.qbittorrent.org/
[sprig]: http://masterminds.github.io/sprig/
[text-template]: https://pkg.go.dev/text/template

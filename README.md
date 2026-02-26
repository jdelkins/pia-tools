# pia-tools

A suite of tools for establishing a [WireGuard][wireguard] tunnel to [Private
Internet Access][PIA] on Linux using [systemd-networkd][], based on [PIA’s REST
API](https://github.com/pia-foss/manual-connections). It can also manage PIA’s
port forwarding feature and optionally notify rTorrent and/or Transmission (via
RPC) of the assigned port; this will allow receiving incoming BitTorrent peer
connections over the VPN.

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

If you use this on a remote firewall, ensure you have out-of-band access before
experimenting with routing.

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
3. Write environment file
4. Install and activate systemd service and timer
5. (Optional) Configure and activate port forwarding service and timer

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
This is "end to edge" encryption, and is the best protection a VPN can provide.
For most users, this is easy and affords plenty of protection, and for that
[PIA][] has nice desktop and mobile apps that are simple to use, but do require
installing them and setting them up. PIA are putting out apps for more and more
devices, but, unfortunately, they can't keep up with all the IoT devices, smart
home hubs, smart TVs, etc. As a result, we have to live with some unencrypted
traffic, and this segment is growing very fast in terms of number of devices and
the number of security issues arising from them.

Another way to use a VPN, available to you only if you control your network
infrastructure, is to set up your LAN's firewall to connect to the VPN service
and route all (or some selection) of the outgoing LAN traffic through the VPN.
This way, any devices on the LAN can benefit from the VPN without configuring
those devices at all. Note that this is "edge to edge" encryption, which is a
bit weaker since there is part of the path, the first part on your LAN, where
traffic is unencrypted. I'm okay with this, because I have physical control
of all apparatus from "end to edge". If you use anyone else's wifi access
point, or connect your access point to someone else's switch, this would not
be the case for you. Stick with the VPN app. This project is for people who,
like me, are okay with this depth-for-breadth tradeoff, and who also use
`systemd-networkd` on their firewall.[^2]

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
and [`pia.network.tmpl`](./systemd/network/pia.network.tmpl) in your
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
| `services.pia-tools.rTorrentUrl`         | `null or string`                    | URL to rTorrent SCGI endpoint.                                                                                                                                                                                                               |
| `services.pia-tools.transmissionUrl`     | `null or string`                    | Transmission RPC endpoint URL. If your Transmission server requires a username and password, set them in `config.services.pia-tools.envFile` with `TRANSMISSION_USERNAME` and `TRANSMISSION_PASSWORD`.                                       |
| `services.pia-tools.envFile`             | `path`                              | **Required.** Path to a file that sets environment variables used to set up the tunnel device. Recognized variables include `PIA_USERNAME` and `PIA_PASSWORD` (required), plus optional `TRANSMISSION_USERNAME` and `TRANSMISSION_PASSWORD`. |
| `services.pia-tools.resetServiceName`    | `string`                            | Name of systemd service for pia-tools tunnel reset.                                                                                                                                                                                          |
| `services.pia-tools.resetTimerConfig`    | `null or systemd timerConfig attrs` | Timer defining frequency of resetting the tunnel. Set to `null` to disable.                                                                                                                                                                  |
| `services.pia-tools.refreshServiceName`  | `string`                            | Name of systemd service for pia-tools tunnel port forwarding refresh.                                                                                                                                                                        |
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
- If the VPN link goes down, LAN traffic will be routed to the WAN instead.
  However, there is a commented configuration block that will change this, if
  you prefer a "kill switch" approach.
- The tunnel gets torn down and remade with a new virtual IP every day at 5:15
  AM.

Note that, theoretically, there is no reason this firewall has to be exposed on
your WAN, it could, for example, be in a DMZ behind your WAN router, with only
certain LAN clients configured (manually or via DHCP) to use its LAN address as
its default route. It just needs WAN egress in order to establish the tunnel.

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
          { pkgs, ... }:
          let
            wgIf = "wg_pia";
            lanIf = "lan";
            transmissionIp = "192.168.100.100";

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

              [Route]
              Destination=0.0.0.0/0
              Gateway={{ .ServerVip }}
              GatewayOnLink=true
              Scope=global
              Table=vpn

              # Look in the main table first for non-default routes
              [RoutingPolicyRule]
              Priority=900
              SuppressPrefixLength=0
              Family=ipv4
              Table=main

              # if the above doesn't match a route from main, and
              # if the traffic is from the lan, then use the vpn
              # as the default route.
              [RoutingPolicyRule]
              Priority=1000
              IncomingInterface=${lanIf}
              Family=ipv4
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

            # If you want an "internet kill switch", i.e. block outgoing lan
            # traffic when vpn is down, you can add something like the following.
            # Without it, LAN traffic is routed out the (unencrypted) default route
            # when the vpn link is down, assuming the firewall rules allow this.
            #
            #systemd.network.networks.lo = {
            #  enable = true;
            #  name = "lo";
            #  routes = [
            #    {
            #      Destination = "0.0.0.0/0";
            #      Metric = 100000;
            #      Type = "unreachable";
            #      Table = "vpn";
            #    }
            #  ];
            #}

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
              # endpoint with /rpc/ added at the end.
              # Ignored unless portForwarding = true
              transmissionUrl = "http://${transmissionIp}:9091/rpc/";

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

### Install the tools

    go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
    go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest     # optional
    go install github.com/jdelkins/pia-tools/cmd/pia-listregions@latest     # optional

### Configure Tunnel Interface

1. Set up `/etc/systemd/network/<interface>.netdev.tmpl` and
   `/etc/systemd/network/<interface>.network.tmpl` template files. These
   templates use the Go package [`text/template`][text-template] to replace
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
    DNS={{ . }}
    {{- end }}

    [Route]
    Destination={{ $gw }}/32
    Scope=link

    {{ range .DnsServers -}}
    [Route]
    Destination={{ . }}/32
    Gateway={{ $gw }}

    {{ end -}}

    [Route]
    Destination=0.0.0.0/0
    Gateway={{ $gw }}
    GatewayOnLink=yes
    ```

    The above examples should be pretty self-explanatory; if not, you should
    read up on `systemd-networkd` and/or [`text/template`][text-template]. For
    info on what other template fields are available (though the above examples
    demonstrate (I think) all of the useful ones), check out [the Tunnel struct
    in the `pia` package](./internal/pia/pia.go#L18). The template processing
    package includes [sprig][], which provides a number of additional template
    functions, should they come in handy.

2. `mkdir /var/cache/pia` and set the directory permissions as restrictive as
   you can, probably `root:root` and mode `0700`. This directory will hold
   `.json` files to cache information including your personal access tokens and
   WireGuard private keys. These files do not store your [PIA][] username or
   password, but should still be treated as private.

3. Run `pia-setup-tunnel --username <user> --password <pass> --if-name <interface>`
   as root to create the `.network` and `.netdev` files corresponding to the
   templates created above.

4. Inspect the generated files. If everything looks okay, run `systemctl
   restart systemd-networkd` as root to reload your network stack and activate
   the tunnel.

That's it, you should have a VPN as your ipv4 default route (assuming you
configured the `.network` file as such). Use something like `curl -4
icanhazip.com` to verify that the ip address is coming from PIA. Every few days
or weeks, repeat steps 3 and 4 to establish a new tunnel. They are good for
some period of time, but I recommend replacing it once a week for privacy
reasons.

### Enabling Port Forwarding

If you also want incoming traffic on a single TCP port and a single UDP port
(both with the same port number) to be forwarded to your VPN endpoint, then
follow the procedure below.

**Note ☞**  The following steps are *not necessary* if you are simply setting
up a tunnel on a gateway. This tool is to enable running a server behind (or
on) the gateway that accepts incoming connections to be accessible through the
VPN.

1. Make sure your WireGuard tunnel to PIA is up and running per the above
   procedure. The following step presumes that you have a working route to
   PIA's WireGuard endpoint virtual IP configured and working. It also assumes
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
   around to implementing a similar feature for [qbittorrent][], the single most
   popular bittorrent server on seedboxes, because I personally don't use it and
   no one has asked, but I believe the RPC capability is there and it should be
   simple to add.)

   If you are running [rtorrent][] or [transmission][] as the server behind the
   gateway, run, respectively, `pia-portforward --if-name <interface> --rtorrent
   http://<rtorrent-ip>:<rtorrent-port>` or `pia-portforward --if-name
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
   IF=pia
   PORT="$(jq -r '.PFSig.port' "/var/cache/pia/${IF}.json")"
   WEBSERVER="192.168.0.80:80"

   nft add table ip nat 2>/dev/null || true
   nft add chain ip nat pia_portforward '{ type nat hook prerouting priority -100 }' 2>/dev/null || true
   nft flush chain ip nat pia_portforward
   nft add rule ip nat pia_portforward tcp dport $PORT dnat to $WEBSERVER
   # add a rule to forward udp traffic if needed
   ```

   ...which would create a nftables rule to forward incoming traffic on your
   assigned port (retrieved from the json cache file) to an internal webserver
   running on `192.168.0.80` at port 80. This example assumes your VPN WireGuard
   interface is named `pia`.
   
   **Note ☞** The example above sets up the mechanism for forwarding
   connections. Assuming you don't run a default-accept firewall (hopefully
   you don't), then to actually permit such connections, you would also have to
   also add a rule to accept them, typically somewhere in the `forward` chain in
   nftables. This is not a lesson in firewall design, so again, you're on your
   own: the possibilities are numerous once you can get the forwarded port number.

4. Every 15 minutes or so (using systemd timers or similar), run `pia-portforward
   --if-name <interface> --refresh` in order to refresh the port forwarding
   assignment using your cached authentication token. If you don't do this,
   PIA will eventually reclaim the port for another customer, and traffic
   meant for you could be delivered somewhere else! You don't want that.
   My testing suggests that PIA is pretty conservative in this area, so
   I think most reasonable failure cases (e.g. your ISP goes down for a
   few hours, so you can't refresh) should be fine, but you definitely want
   to keep refreshing the assignment as long as you're intending to use it.

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

| Flag                         | Environment Var | Default          | Meaning                                                                                         |
|------------------------------|-----------------|------------------|-------------------------------------------------------------------------------------------------|
| `--region string`            | PIA_REGION      | `auto`           | PIA region identifier (e.g., us_chicago, us_texas)                                              |
| `--username string`          | PIA_USERNAME    | _required_       | PIA account username                                                                            |
| `--password string`          | PIA_PASSWORD    | _required_       | PIA account password                                                                            |
| `--if-name string`           | _n/a_           | `pia`            | Interface name to create or reconfigure (e.g., v4, wg0)                                         |
| `--netdev-file key=value,…`  | _n/a_           | _see below_      | Write a .netdev file using a key/value specification                                            |
| `--network-file key=value,…` | _n/a_           | _see below_      | Write a .network file using a key/value specification                                           |
| `--cache-dir`                | _n/a_           | `/var/cache/pia` | directory in which to save a json file with the tunnel parameters.                              |
| `--wg-binary`                | _n/a_           | `wg`             | path to the `wg` binary from wireguard-tools (look in $PATH by default)                         |
| `--from-cache`               | _n/a_           | _unset_          | Skip accessing PIA's api, and just (re-)generate the networkd files. Useful to debug templates. |

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

| Flag                             | Environment Var       | Default | Meaning                                                                  |
|----------------------------------|-----------------------|---------|--------------------------------------------------------------------------|
| `--username string`              | PIA_USERNAME          | _none_  | Used to get a new authentication token, if expired.                      |
| `--password string`              | PIA_PASSWORD          | _none_  | ibid                                                                     |
| `--if-name string`               | _n/a_                 | `pia`   | Interface name associated with the active PIA tunnel                     |
| `--rtorrent string`              | RTORRENT              | _none_  | rTorrent SCGI endpoint (e.g., 127.0.0.1:5000)                            |
| `--transmission string`          | TRANSMISSION          | _none_  | Transmission RPC endpoint (e.g., http://localhost:9091/transmission/rpc) |
| `--transmission-username string` | TRANSMISSION_USERNAME | _none_  | Transmission RPC username (if required)                                  |
| `--transmission-password string` | TRANSMISSION_PASSWORD | _none_  | Transmission RPC password (if required)                                  |
| `--refresh`                      | _n/a_                 | _unset_ | Don't get a new port forwarding assignment, just refresh the active one  |

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
export TRANSMISSION=http://192.168.77.77:9091/transmission/rpc
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

#### Typical Workflow

1. Run `pia-setup-tunnel` to (re-)configure the WireGuard interface files.
2. Bring the interface up using systemd-networkd.
3. Run `pia-portforward` to retrieve and apply the forwarded port to downstream services.
4. Schedule `pia-portforward` periodically to renew the port forwarding lease.

Both commands are designed to be automation-friendly and safe for use within
systemd units like the included examples in the [systemd](./systemd) directory.

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

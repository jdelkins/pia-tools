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

## Install

    go install github.com/jdelkins/pia-tools@latest


[systemd-networkd]: https://www.freedesktop.org/software/systemd/man/systemd.network.html
[wireguard]: https://www.wireguard.com/
[PIA]: https://www.privateinternetaccess.com/
[rtorrent]: https://github.com/rakshasa/rtorrent

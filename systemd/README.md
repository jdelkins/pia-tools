# pia-tools: systemd components

Contains configuration elements for using [Private Internet Access][PIA] wireguard tunnels with
**systemd-networkd**.

Assumes:

- You have an account with [PIA][]
- You use **systemd-networkd** for your network stack
- You have installed the accompanying go tools:

        $ GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-setup-tunnel@latest
        $ GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-portforward@latest

[PIA]: https://privateinternetaccess.com

## Installation

1. Clone the repo and copy or link the files into place

        $ somewhere=~/Code/pia-tools
        $ git clone https://github.com/jdelkins/pia-tools $somewhere
        $ ln -s $somewhere/systemd/system/*.{timer,service} /etc/systemd/system/
        $ cp $somewhere/systemd/network/*.tmpl /etc/systemd/network/
        $ cp $somewhere/systemd/pia.conf /etc/pia.conf
        $ mkdir /var/cache/pia/

2. Edit `/etc/pia.conf` and set variables according to your setup.

3. Edit the `/etc/systemd/network/*.tmpl` files to define how your tunnel will
   be configured. The provided template files set a default route through the
   tunnel, and add PIA's provided DNS servers to **systemd-resolved**. This may
   or may not be what you want; if not, edit them. If you can't figure out the
   template language, read up on go's `text/template` package.

4. If you wish, adjust the `.timer` files to change the timing of when the
   scripts are run.

5. If you don't wish to use the rtorrent notification feature, edit the
   `.service` files and remove the `-rtorrent $RTORRENT_URL` parts.

6. Enable the timers:

        $ systemctl enable --now pia-reset-tunnel.timer
        $ systemctl enable --now pia-pf-refresh.timer

7. If you don't wish to use the port forwarding setup, then you don't need
   `pia-pf-refresh.timer`. In this case, you might want to also edit
   `pia-reset-tunnel.service` since it also reconfigures the forwarded port
   after the new tunnel comes back up.

8. If you want to get your tunnel up now, run the reset service by hand:

        $ systemctl start pia-reset-tunnel.service

9. Test your setup:

        $ ping 10.0.0.242

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

    <pre><code>
    $ <i>repo</i>=~/Code/pia-tools
    $ git clone https://github.com/jdelkins/pia-tools $<i>repo</i>
    $ ln -s $<i>repo</i>/systemd/system/*.{timer,service} /etc/systemd/system/
    $ cp $<i>repo</i>/systemd/network/*.tmpl /etc/systemd/network/
    $ cp $<i>repo</i>/systemd/pia.conf /etc/pia.conf
    $ mkdir /var/cache/pia/
    </code></pre>

2. Edit `/etc/pia.conf` and set variables according to your setup. For
   `PIA_REGION` you can use the included `pia-listregions` utility to choose
   a region id with a low ping time and the desired feature set, or, simply
   specify the special region "auto" to select the region with the lowest ping
   time.

   To use the `pia-listregions` tool, install then run it like this:

        $ GOBIN=/usr/local/bin go install github.com/jdelkins/pia-tools/cmd/pia-listregions@latest
        $ /usr/local/bin/pia-listregions
        ID                 NAME                         PING       WG?  PF?
        ==============     =======================      =========  ===  ===
        us_houston         US Houston                   25 ms       ✓
        us_south_west      US Texas                     30 ms       ✓
        us_atlanta         US Atlanta                   38 ms       ✓
        us_denver          US Denver                    39 ms       ✓
        bogota             Colombia                     46 ms       ✓    ✓
        santiago           Chile                        46 ms       ✓    ✓
        us_florida         US Florida                   47 ms       ✓
        ar                 Argentina                    48 ms       ✓    ✓
        ...

3. Edit the `/etc/systemd/network/*.tmpl` files to define how your tunnel will
   be configured. The provided template files set a default route through the
   tunnel, and add PIA's provided DNS servers to **systemd-resolved**. This may
   or may not be what you want; if not, edit them. If you can't figure out the
   template language, read up on go's `text/template` package.

4. If you wish, adjust the `.timer` files to change the timing of when the
   scripts are run.

5. If you don't wish to use the rtorrent notification feature, edit the
   `.service` files and remove the `-rtorrent $RTORRENT_URL` parts.

6. Enable the timers. Use whatever interface name you want in place of `wgpia0`.

        $ systemctl enable --now pia-reset-tunnel@wgpia0.timer
        $ systemctl enable --now pia-pf-refresh@wgpia0.timer

7. If you don't wish to use the port forwarding setup, then you don't need
   `pia-pf-refresh@.timer`. In this case, you might also want to also edit
   `pia-reset-tunnel@.service` (e.g. `systemctl edit pia-tunnel-reset@wgpia0.service`)
   since it also reconfigures the forwarded port after the new tunnel comes
   back up. If you don't install the `pia-portfward` binary, the port
   forwarding configuration will fail harmlessly.

8. If you want to get your tunnel up now, run the reset service by hand as
   follows. This service first tears down the existing configuration (as it is
   intended to reconfigure the tunnel with a completely new private key and,
   hopefully, a different public ip address), but if the interface doesn't
   already exist, the tear-down steps will fail harmlessly.

        $ systemctl start pia-reset-tunnel@wgpia0.service

9. Test your setup:

        $ ping 10.0.0.242

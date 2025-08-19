{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:

    flake-utils.lib.eachDefaultSystem (
      system:
      let
        inherit (nixpkgs) lib;
        pkgs = nixpkgs.legacyPackages.${system};

        pkg = pkgs.buildGoModule {
          pname = "pia-tools";
          version = "1.2.0";
          src = ./.;
          vendorHash = "sha256-QJ5KOZ/SLfOk6A/vQR4RK7OsNGbwB8nxC37YCz3Xy+w=";
          meta = {
            description = "Toolset to manage wireguard tunnels to privateinternetaccess.com";
            homepage = "https://github.com/jdelkins/pia-tools";
            license = lib.licenses.mit;
          };
        };

        module =
          with lib;
          with options;
          {
            config,
            pkgs,
            utils,
            ...
          }:
          let
            cfg = config.pia-tools;
            whitelist-sh = pkgs.writeShellScript "pia-whitelist-${cfg.ifname}.sh" ''
              ip=$(${pkgs.jq}/bin/jq -r .server_ip </var/cache/pia/${cfg.ifname}.json)
              ${pkgs.nftables}/bin/nft add element ${cfg.whitelistSet} "{$ip}" && echo "Whitelisted $ip"
            '';
          in
          {
            options.pia-tools = {
              enable = mkEnableOption "pia-tools";

              user = mkOption {
                description = "User to run the tool as";
                type = types.str;
                default = "pia";
              };

              group = mkOption {
                description = "Group to run the tool as";
                type = types.str;
                default = "pia";
              };

              ifname = mkOption {
                description = "Name of PIA WireGuard network interface";
                type = types.str;
                default = "wg_pia";
              };

              region = mkOption {
                description = "Region to connect to, or auto by default";
                type = types.str;
                default = "auto";
                example = "ca_toronto";
              };

              rTorrentParams = mkOption {
                description = "Additional parameters to pia-setup-tunnel to connect to rTorrent";
                type = types.str;
                default = "";
                example = "--rtorrent https://rtorrent.local";
              };

              transmissionParams = mkOption {
                description = "Additional parameters to pia-setup-tunnel to connect to Transmission bittorrent server";
                type = types.str;
                example = "--transmission 192.168.1.20 --transmission-username $TRANSMISSION_USERNAME --transmission-password $TRANSMISSION_PASSWORD";
                default = "";
              };

              envFile = mkOption {
                description = ''
                  Required. Path to file setting environment variables to be used
                  in setting up the tunnel device ${cfg.ifname}.
                  The recognized variables are as follows. Read accompanying documentation
                  and/or use ``pia-setup-tunnel --help``.

                     PIA_USERNAME (required)
                     PIA_PASSWORD (required)
                '';
                type = types.path;
              };

              resetServiceName = mkOption {
                description = "Name of systemd service for pia-tools tunnel reset";
                type = types.str;
                default = "pia-reset-${cfg.ifname}";
              };
              resetTimerConfig = mkOption {
                description = "Timer defining frequency of resetting the tunnel. Set to null to disable.";
                type = with types; nullOr (attrsOf utils.systemdUtils.unitOptions.unitOption);
                example = "null";
                default = {
                  OnCalendar = "Wed *-*-* 03:00:00";
                  RandomizedDelaySec = "72h";
                };
              };

              refreshServiceName = mkOption {
                description = "Name of systemd service for pia-tools tunnel port forwarding refresh";
                type = types.str;
                default = "pia-pf-refresh-${cfg.ifname}";
              };
              refreshTimerConfig = mkOption {
                description = "Timer defining frequency of refreshing the tunnel's port forwarding assignment. Set to null to disable.";
                type = with types; nullOr (attrsOf utils.systemdUtils.unitOptions.unitOption);
                example = "null";
                default = {
                  OnCalendar = "*-*-* *:00/15:00";
                };
              };

              whitelistSet = mkOption {
                description = "nftables set to which to add the configured server's ip after resetting";
                type = with types; nullOr str;
                example = "inet filter whitelist_4";
              };

              portForwarding = mkOption {
                description = "whether to request a port forwarding assignment from PIA";
                type = types.bool;
                default = false;
              };

              netdevTemplateFile = mkOption {
                description = "systemd.netdev file containing template parameters with which to generate the actual netdev";
                type = types.path;
                default = ./systemd/network/pia.netdev.tmpl;
                example = "/etc/systemd/network/pia.netdev.tmpl";
              };

              networkTemplateFile = mkOption {
                description = "systemd.network file containing template parameters with which to generate the actual network";
                type = types.path;
                default = ./systemd/network/pia.network.tmpl;
                example = "/etc/systemd/network/pia.network.tmpl";
              };
            };

            config = lib.mkIf cfg.enable {
              users.users.pia = lib.mkIf (cfg.user == "pia") {
                description = "pia-tools system user account";
                isSystemUser = true;
                group = cfg.group;
              };
              users.groups.pia = lib.mkIf (cfg.group == "pia") {
                members = [ cfg.user ];
              };
              systemd.tmpfiles.settings."50-pia-${cfg.ifname}" = {
                "/var/cache/pia".d = {
                  user = cfg.user;
                  group = cfg.group;
                  mode = "0750";
                };
                "/etc/systemd/network/${cfg.ifname}.netdev".f = {
                  user = cfg.user;
                  group = config.users.groups.systemd-network.name;
                  mode = "640";
                };
                "/etc/systemd/network/${cfg.ifname}.network".f = {
                  user = cfg.user;
                  group = config.users.groups.systemd-network.name;
                  mode = "640";
                };
              };

              # Tunnel reset service and timer
              systemd.services.${cfg.resetServiceName} = {
                description = "Reset the ${cfg.ifname} VPN tunnel";
                name = "${cfg.resetServiceName}.service";
                path = [ pkgs.wireguard-tools ];
                serviceConfig = {
                  User = cfg.user;
                  Type = "oneshot";
                  # username and password are passed in via environment variables PIA_USERNAME and PIA_PASSWORD, respectively
                  EnvironmentFile = cfg.envFile;
                  PassEnvironment = "PIA_USERNAME PIA_PASSWORD";
                  ExecStart = ''${pkg}/bin/pia-setup-tunnel --region ${cfg.region} --ifname ${cfg.ifname} --netdev-template "${cfg.netdevTemplateFile}" --network-template "${cfg.networkTemplateFile}"'';
                  ExecStartPost = [
                    "-${pkgs.iproute2}/bin/ip link set down dev ${cfg.ifname}"
                    "-${pkgs.iproute2}/bin/ip link del ${cfg.ifname}"
                    "${pkgs.systemd}/bin/networkctl reload"
                    "${pkgs.systemd}/bin/networkctl reconfigure ${cfg.ifname}"
                    "${pkgs.systemd}/bin/networkctl up ${cfg.ifname}"
                  ]
                  ++ lib.optionals (cfg.whitelistSet != null) [
                    "+${whitelist-sh}"
                  ]
                  ++ lib.optionals (cfg.portForwarding) [
                    "${pkgs.coreutils}/bin/sleep 10"
                    "${pkg}/bin/pia-portforward --ifname ${cfg.ifname} ${cfg.rTorrentParams} ${cfg.transmissionParams}"
                  ];
                };
              };
              systemd.timers.${cfg.resetServiceName} = lib.mkIf (cfg.resetTimerConfig != null) {
                description = "Reset the ${cfg.ifname} VPN tunnel";
                name = "${cfg.resetServiceName}.timer";
                timerConfig = cfg.resetTimerConfig;
                wantedBy = [ "timers.target" ];
              };

              # Port forwarding refresh service and timer
              systemd.services.${cfg.refreshServiceName} = lib.mkIf (cfg.portForwarding) {
                description = "Refresh port forwarding assignment for the ${cfg.ifname} VPN tunnel";
                name = "${cfg.refreshServiceName}.service";
                path = [ pkgs.wireguard-tools ];
                serviceConfig = {
                  User = cfg.user;
                  Type = "oneshot";
                  EnvironmentFile = cfg.envFile;
                  PassEnvironment = "PIA_USERNAME PIA_PASSWORD";
                  ExecStart = "${pkg}/bin/pia-portforward --ifname ${cfg.ifname} --refresh ${cfg.rTorrentParams} ${cfg.transmissionParams}";
                }
                // lib.attrsets.optionalAttrs (cfg.whitelistSet != null) {
                  ExecStartPost = "+${whitelist-sh}";
                };
              };
              systemd.timers.${cfg.refreshServiceName} =
                lib.mkIf (cfg.portForwarding && cfg.refreshTimerConfig != null)
                  {
                    description = "Refresh port forwarding assignment for the ${cfg.ifname} VPN tunnel";
                    name = "${cfg.refreshServiceName}.timer";
                    timerConfig = cfg.refreshTimerConfig;
                    wantedBy = [ "timers.target" ];
                  };

              environment.systemPackages = [ pkg ];
            };
          };
      in
      {
        packages = {
          pia-tools = pkg;
          default = pkg;
        };
        nixosModules = {
          pia-tools = module;
          default = module;
        };
      }
    );

}

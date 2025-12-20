{
  pia-tools ? null,
}:
{
  config,
  lib,
  pkgs,
  utils,
  ...
}:

let
  cfg = config.pia-tools;
  cacheFile = "${cfg.cacheDir}/${cfg.ifname}.json";
  cacheNetdev = "${cfg.cacheDir}/${cfg.ifname}.netdev";
  cacheNetwork = "${cfg.cacheDir}/${cfg.ifname}.network";

  inherit (lib)
    mkOption
    types
    ;

  inherit (lib.options)
    mkEnableOption
    ;

  getIp = "${pkgs.jq}/bin/jq -r .server_ip <${cacheFile}";
in
{
  options.pia-tools = {
    enable = mkEnableOption "pia-tools";

    package = mkOption {
      description = "The pia-tools package to use";
      type = types.package;
      default = pia-tools;
    };

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

    cacheDir = mkOption {
      description = "Where to store tunnel descriptions in json format, containing private keys.";
      type = types.path;
      example = "/run/pia";
      default = "/var/cache/pia";
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
      description = "Name of systemd service for pia-tools tunnel port forwarding refresh.";
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

    whitelistScript = mkOption {
      description = ''
        Script to run when the wireguard endpoint is established, ostensibly to add the ip to a firewall passlist.
        The script will be called with the ip as the only argument. Set to null to ignore.
      '';
      type = with types; nullOr path;
      default = null;
      example = lib.literalExpression ''
        pkgs.writeShellScript "whitelist_ip" '''
          ''${pkgs.nftables}/bin/nft add element inet filter passlist "{ $1 }"
        '''
      '';
    };

    portForwarding = mkOption {
      description = "whether to request a port forwarding assignment from PIA.";
      type = types.bool;
      default = false;
    };

    netdevTemplateFile = mkOption {
      description = "systemd.netdev file containing template parameters with which to generate the actual netdev.";
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

    netdevFile = mkOption {
      description = "systemd.netdev file path, specifying the location to install the generated netdev.";
      type = types.path;
      default = "/etc/systemd/network/10-${cfg.ifname}.netdev";
      example = "/etc/systemd/network/10-pia.netdev";
    };

    networkFile = mkOption {
      description = "systemd.network file path, specifying the location to install the generated network.";
      type = types.path;
      default = "/etc/systemd/network/40-${cfg.ifname}.network";
      example = "/etc/systemd/network/40-pia.network";
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
    systemd.tmpfiles.settings."50-pia-${cfg.ifname}" =
      let
        mk = type: group: {
          ${type} = {
            inherit group;
            inherit (cfg) user;
            mode = if type == "d" then "0750" else "0640";
          };
        };
      in
      {
        ${cfg.cacheDir} = mk "d" cfg.group;
        ${cfg.netdevFile} = mk "f" config.users.groups.systemd-network.name;
        ${cfg.networkFile} = mk "f" config.users.groups.systemd-network.name;
      };

    # Tunnel reset service and timer
    systemd.services.${cfg.resetServiceName} = {
      description = "Reset the ${cfg.ifname} VPN tunnel";
      name = "${cfg.resetServiceName}.service";
      path = [ cfg.package ];
      serviceConfig = {
        User = cfg.user;
        Type = "oneshot";
        ReadWritePaths = lib.unique [
          (builtins.dirOf cfg.netdevFile)
          (builtins.dirOf cfg.networkFile)
          cfg.cacheDir
        ];
        ReadOnlyPaths = [ "/nix/store" ];
        # username and password are passed in via environment variables PIA_USERNAME and PIA_PASSWORD, respectively
        EnvironmentFile = cfg.envFile;
        PassEnvironment = "PIA_USERNAME PIA_PASSWORD";
        ExecStart = ''${cfg.package}/bin/pia-setup-tunnel --wg-binary ${pkgs.wireguard-tools}/bin/wg --cachedir ${cfg.cacheDir} --region ${cfg.region} --ifname ${cfg.ifname} --netdev-template "${cfg.netdevTemplateFile}" --netdev "${cacheNetdev}" --network-template "${cfg.networkTemplateFile}" --network "${cacheNetwork}"'';
        ExecStartPost = [
          ''+${pkgs.coreutils}/bin/install -o systemd-network -g systemd-network -m 0440 "${cacheNetdev}" "${cfg.netdevFile}"''
          ''+${pkgs.coreutils}/bin/install -o root -g root -m 0444 "${cacheNetwork}" "${cfg.networkFile}"''
          "+-${pkgs.iproute2}/bin/ip link set down dev ${cfg.ifname}"
          "+-${pkgs.iproute2}/bin/ip link del ${cfg.ifname}"
          "+${pkgs.systemd}/bin/networkctl reload"
          "+${pkgs.systemd}/bin/networkctl reconfigure ${cfg.ifname}"
          "+${pkgs.systemd}/bin/networkctl up ${cfg.ifname}"
        ]
        ++ lib.optionals (cfg.whitelistScript != null) [
          ''+${pkgs.bash}/bin/bash -c '${cfg.whitelistScript} "$(${getIp})"' ''
        ]
        ++ lib.optionals (cfg.portForwarding) [
          "${pkgs.coreutils}/bin/sleep 10"
          "${cfg.package}/bin/pia-portforward --cachedir ${cfg.cacheDir} --ifname ${cfg.ifname} ${cfg.rTorrentParams} ${cfg.transmissionParams}"
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
        ExecStart = "${cfg.package}/bin/pia-portforward --cachedir ${cfg.cacheDir} --ifname ${cfg.ifname} --refresh ${cfg.rTorrentParams} ${cfg.transmissionParams}";
      }
      // lib.attrsets.optionalAttrs (cfg.whitelistScript != null) {
        ExecStartPost = ''+${pkgs.bash}/bin/bash -c '${cfg.whitelistScript} "$(${getIp})"' '';
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

    environment.systemPackages = [ cfg.package ];
  };
}

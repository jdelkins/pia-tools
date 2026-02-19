{
  config,
  lib,
  pkgs,
  utils,
  ...
}:

let
  cfg = config.services.pia-tools;
  cacheFile = "${cfg.cacheDir}/${cfg.ifname}.json";

  inherit (lib)
    mkOption
    types
    ;

  inherit (lib.options)
    mkEnableOption
    ;

  getIp = ''${pkgs.jq}/bin/jq -r .server_ip <${cacheFile} | ${pkgs.coreutils}/bin/tr -d \\n'';

  serviceEnvFile = pkgs.writeText "service_params.sh" ''
    ${lib.optionalString (cfg.transmissionUrl != null) "TRANSMISSION=${cfg.transmissionUrl}"}
    ${lib.optionalString (cfg.rTorrentUrl != null) "RTORRENT=${cfg.rTorrentUrl}"}
  '';
in
{
  options.services.pia-tools = {
    enable = mkEnableOption "pia-tools";

    package = mkOption {
      description = "The pia-tools package to use";
      type = types.package;
      default = pkgs.callPackage ./package.nix { };
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

    rTorrentUrl = mkOption {
      description = "URL to rTorrent SCGI endpoint";
      type = types.nullOr types.str;
      default = null;
      example = "https://rtorrent.local";
    };

    transmissionUrl = mkOption {
      description = ''
        Transmission RPC endpoint URL. If your transmission server requires
        a username and password, set them in config.services.pia-tools.envFile, with
        the variables TRANSMISSION_USERNAME and TRANSMISSION_PASSWORD.
      '';
      type = types.nullOr types.str;
      default = null;
      example = "http://192.168.100.100:9091/rpc/";
    };

    envFile = mkOption {
      description = ''
        Required. Path to file setting environment variables to be used
        in setting up the tunnel device ${cfg.ifname}.
        The recognized variables are as follows. Read accompanying documentation
        and/or use ``pia-setup-tunnel --help``.

           PIA_USERNAME (required)
           PIA_PASSWORD (required)

        Optional, if needed:

           TRANSMISSION_USERNAME
           TRANSMISSION_PASSWORD
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
        EnvironmentFile = [
          serviceEnvFile
          cfg.envFile
        ];
        UMask = "0002";
        CapabilityBoundingSet = [
          "CAP_NET_ADMIN"
        ];
        AmbientCapabilities = [
          "CAP_NET_ADMIN"
        ];
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectKernelLogs = true;
        ProtectControlGroups = true;
        NoNewPrivileges = true;
        RestrictSUIDSGID = true;
        LockPersonality = true;
        RemoveIPC = true;
        ProtectClock = true;
        ProtectHostname = true;
        ProtectProc = "invisible";
        ProcSubset = "pid";
        RestrictRealtime = true;
        RestrictNamespaces = true;
        MemoryDenyWriteExecute = true;
        SystemCallArchitectures = "native";
        SystemCallFilter = [
          "~@clock"
          "~@cpu-emulation"
          "~@debug"
          "~@module"
          "~@mount"
          "~@obsolete"
          "~@raw-io"
          "~@reboot"
          "~@swap"
          "~@resources"
        ];
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
          "AF_NETLINK"
        ];
        ExecStart = ''+${cfg.package}/bin/pia-setup-tunnel --wg-binary ${pkgs.wireguard-tools}/bin/wg --cache-dir ${cfg.cacheDir} --region ${cfg.region} --if-name ${cfg.ifname} --netdev-file="template=${cfg.netdevTemplateFile},output=${cfg.netdevFile},group=systemd-network,mode=0440" --network-file="template=${cfg.networkTemplateFile},output=${cfg.networkFile},mode=0444"'';
        ExecStartPost = [
          "-${pkgs.iproute2}/bin/ip link set down dev ${cfg.ifname}"
          "-${pkgs.iproute2}/bin/ip link del ${cfg.ifname}"
          "+${pkgs.systemd}/bin/networkctl reload"
          "+${pkgs.systemd}/bin/networkctl reconfigure ${cfg.ifname}"
          "+${pkgs.systemd}/bin/networkctl up ${cfg.ifname}"
        ]
        ++ lib.optionals (cfg.whitelistScript != null) [
          ''${pkgs.bash}/bin/bash -c '${cfg.whitelistScript} "$(${getIp})"' ''
        ]
        ++ lib.optionals (cfg.portForwarding) [
          "${pkgs.coreutils}/bin/sleep 10"
          "-${cfg.package}/bin/pia-portforward --cache-dir ${cfg.cacheDir} --if-name ${cfg.ifname}"
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
        EnvironmentFile = [
          serviceEnvFile
          cfg.envFile
        ];
        ExecStart = "${cfg.package}/bin/pia-portforward --cache-dir ${cfg.cacheDir} --if-name ${cfg.ifname} --refresh";
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

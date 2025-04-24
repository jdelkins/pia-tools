{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { nixpkgs, ... }: let
    inherit (nixpkgs) lib;
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};

    pkg = pkgs.buildGoModule {
      pname = "pia-tools";
      version = "1.1.6";
      src = ./.;
      vendorHash = "sha256-V/yJl12zPhG/e0IeUzFiOrJTtm9Kl/MEb2v2xV7xzAI=";
      meta = {
        description = "Toolset to manage wireguard tunnels to privateinternetaccess.com";
        homepage = "https://github.com/jdelkins/pia-tools";
        license = lib.licenses.mit;
      };
    };

    module = with lib; with options; { config, pkgs, utils, ...}: let
      cfg = config.pia-tools;
    in {
      options.pia-tools = {
        enable = mkEnableOption "pia-tools";
        user = mkOption {
          description = "User to run the tool as.";
          type = types.str;
          default = "pia";
        };
        group = mkOption {
          description = "Group to run the tool as.";
          type = types.str;
          default = "pia";
        };
        ifname = mkOption {
          description = "Name of PIA WireGuard network interface";
          type = types.str;
          default = "wg_pia";
        };
        region = mkOption {
          description = "Region to connect to, or auto by default.";
          type = types.str;
          default = "auto";
        };
        rTorrentParams = mkOption {
          description = "Additional parameters to pia-setup-tunnel to connect to rTorrent";
          type = types.str;
          default = "";
        };
        transmissionParams = mkOption {
          description = "Additional parameters to pia-setup-tunnel to connect to Transmission bittorrent server";
          type = types.str;
          default = "";
        };
        envFile = mkOption {
          description = ''
            Required. Path to file setting environment variables to be used
            in setting up the tunnel device ${cfg.ifname}.
            The recognized variables are as follows. Read accompanying documentation
            and/or use ``${pkg}/bin/pia-setup-tunnel --help``.

               PIA_USERNAME (required)
               PIA_PASSWORD (required)
          '';
          type = types.path;
        };
        serviceName = mkOption {
          description = "Name of systemd service for pia-tools tunnel refresh";
          type = types.str;
          default = "pia-refresh-${cfg.ifname}";
        };
        timerConfig = mkOption {
          description = "Timer defining frequency of resetting the tunnel. Set to null to disable.";
          type = with types; nullOr (attrsOf utils.systemdUtils.unitOptions.unitOption);
          example = "null";
          default = {
            OnCalendar = "Wed *-*-* 03:00:00";
            RandomizedDelaySec = "72h";
          };
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
        systemd.tmpfiles.settings."50-pia"."/var/cache/pia".d = {
          user = cfg.user;
          group = cfg.group;
          mode = "0750";
        };
        systemd.services.${cfg.serviceName} = {
          name = "${cfg.serviceName}.service";
          serviceConfig = {
            Type = "oneshot";
            EnvironmentFile = cfg.envFile;
            ExecStart = "${pkg}/bin/pia-setup-tunnel --region ${cfg.region} --username $PIA_USERNAME --password $PIA_PASSWORD --ifname ${cfg.ifname}";
            ExecStartPost = [
              "-${pkgs.iproute2}/bin/ip link set down dev ${cfg.ifname}"
              "-${pkgs.iproute2}/bin/ip link del ${cfg.ifname}"
              "${pkgs.systemd}/bin/networkctl reload"
              "${pkgs.systemd}/bin/networkctl reconfigure ${cfg.ifname}"
              "${pkgs.systemd}/bin/networkctl up ${cfg.ifname}"
              "${pkgs.coreutils}/bin/sleep 10"
              "-${pkg}/bin/pia-portforward --ifname ${cfg.ifname} ${cfg.rTorrentParams} ${cfg.transmissionParams}"
            ];
          };
        };
        systemd.timers.${cfg.serviceName} = lib.mkIf (cfg.timerConfig != null) {
          name = "${cfg.serviceName}.timer";
          timerConfig = cfg.timerConfig;
          wantedBy = [ "timers.target" ];
        };
      };
    };
  in {
    packages.${system} = {
      pia-tools = pkg;
      default = pkg;
    };
    nixosModules = {
      pia-tools = module;
      default = module;
    };
  };
}

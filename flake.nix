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
    module = with lib; with options; { config, pkgs, utils, ...}: {
      options.pia-tools = {
        enable = mkEnableOption "pia-tools";
        user = mkOption {
          description = "User to run the tool as.";
          type = types.string;
          default = "pia";
        };
        group = mkOption {
          description = "Group to run the tool as.";
          type = types.string;
          default = "pia";
        };
        ifname = mkOption {
          description = "Name of PIA WireGuard network interface";
          type = types.string;
          default = "wg_pia";
        };
        region = mkOption {
          description = "Region to connect to, or auto by default.";
          type = types.string;
          default = "auto";
        };
        rTorrentParams = mkOption {
          description = "Additional parameters to pia-setup-tunnel to connect to rTorrent";
          type = types.string;
          default = "";
        };
        transmissionParams = mkOption {
          description = "Additional parameters to pia-setup-tunnel to connect to Transmission bittorrent server";
          type = types.string;
          default = "";
        };
        timerEnabled = mkOption {
          description = "Whether to enable a timer to periodically refresh the tunnel.";
          type = types.bool;
          default = true;
        };
        envFile = mkOption {
          description = ''
            Required. Path to file setting environment variables to be used
            in setting up the tunnel device ${config.pia-tools.ifname}.
            The recognized variables are as follows. Read accompanying documentation
            and/or use ``${pkg}/bin/pia-setup-tunnel --help``.

               PIA_USERNAME (required)
               PIA_PASSWORD (required)
          '';
          type = types.path;
        };
        serviceName = mkOption {
          description = "Name of systemd service for pia-tools tunnel refresh";
          type = types.string;
          default = "pia-refresh-${config.pia-tools.ifname}";
        };
        timerConfig = mkOption {
          description = "Timer defining frequency of resetting the tunnel.";
          type = utils.systemdUtils.types.timerConfig;
          default = {
            OnCalendar = "Wed *-*-* 03:00:00";
            RandomizedDelaySec = "72h";
          };
        };
      };

      config = lib.mkIf config.pia-tools.enable {
        users.users.pia = lib.mkIf (config.pia-tools.user == "pia") {
          description = "pia-tools system user account";
          isSystemUser = true;
          group = config.pia-tools.group;
        };
        users.groups.pia = lib.mkIf (config.pia-tools.group == "pia") {
          members = [ config.pia-tools.user ];
        };
        systemd.tmpfiles.settings."50-pia"."/var/cache/pia".d = {
          user = config.pia-tools.user;
          group = config.pia-tools.group;
          mode = "0750";
        };
        systemd.services.${config.pia-tools.serviceName} = {
          name = "${config.pia-tools.serviceName}.service";
          serviceConfig = {
            Type = "oneshot";
            EnvironmentFile = config.pia-tools.envFile;
            ExecStart = "ExecStart=${pkg}/bin/pia-setup-tunnel --region ${config.pia-tools.region} --username $PIA_USERNAME --password $PIA_PASSWORD --ifname ${config.pia-tools.ifname}";
            ExecStartPost = [
              "-${pkgs.iproute2}/bin/ip link set down dev ${config.pia-tools.ifname}"
              "-${pkgs.iproute2}/bin/ip link del ${config.pia-tools.ifname}"
              "${pkgs.systemd}/bin/networkctl reload"
              "${pkgs.systemd}/bin/networkctl reconfigure ${config.pia-tools.ifname}"
              "${pkgs.systemd}/bin/networkctl up ${config.pia-tools.ifname}"
              "${pkgs.coreutils}/bin/sleep 10"
              "-${pkg}/bin/pia-portforward --ifname ${config.pia-tools.ifname} ${config.pia-tools.rTorrentParams} ${config.pia-tools.transmissionParams}"
            ];
          };
        };
        systemd.timers.${config.pia-tools.serviceName} = lib.mkIf config.pia-tools.timerEnabled {
          name = "${config.pia-tools.serviceName}.timer";
          timerConfig = config.pia-tools.timerConfig;
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

{ pkgs, lib, ... }:

pkgs.buildGoModule {
  pname = "pia-tools";
  version = "2.0.0";
  src = ./.;
  vendorHash = "sha256-CwNP6jQuDUPK8lHGEQIHlSzcmmT6KhbUhY3iYsO/eVI=";
  env.CGO_ENABLED = 0;
  meta = {
    description = "Toolset to manage wireguard tunnels to privateinternetaccess.com";
    homepage = "https://github.com/jdelkins/pia-tools";
    license = lib.licenses.mit;
  };
}

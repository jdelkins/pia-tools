{ pkgs, lib, ... }:

pkgs.buildGoModule {
  pname = "pia-tools";
  version = "2.1.0";
  src = ./.;
  vendorHash = "sha256-6WdBQylihIMCliB1wwJqkAVSbSLCq0p332+10Tff8ss=";
  env.CGO_ENABLED = 0;
  meta = {
    description = "Toolset to manage wireguard tunnels to privateinternetaccess.com";
    homepage = "https://github.com/jdelkins/pia-tools";
    license = lib.licenses.mit;
  };
}

{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    let
      inherit (nixpkgs) lib;
    in
    lib.recursiveUpdate
      {
        nixosModules = rec {
          pia-tools = import ./module.nix;
          default = pia-tools;
        };
      }
      (
        flake-utils.lib.eachDefaultSystem (
          system:
          let
            pkgs = nixpkgs.legacyPackages.${system};
            pkg = pkgs.callPackage ./package.nix { };
          in
          {
            apps = rec {
              default = listregions;
              listregions = {
                type = "app";
                program = "${pkg}/bin/pia-listregions";
              };
              setup-tunnel = {
                type = "app";
                program = "${pkg}/bin/pia-setup-tunnel";
              };
              portforward = {
                type = "app";
                program = "${pkg}/bin/pia-portforward";
              };
            };

            packages = {
              pia-tools = pkg;
              default = pkg;
            };

            devShells = {
              default = pkgs.mkShell {
                packages = with pkgs; [
                  go               # compiler & toolchain
                  gopls            # language server
                  delve            # debugger (dlv)
                  golangci-lint    # linter aggregator
                  golangci-lint-langserver
                  wireguard-tools  # handy for testing the app’s domain
                  # goreleaser
                ];

                shellHook = ''
                  echo "pia-tools dev shell — $(go version)"
                  export CGO_ENABLED=0
                '';
              };
            };
          }
        )
      );

}

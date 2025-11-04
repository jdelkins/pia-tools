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
          pia-tools = import ./module.nix { };
          default = pia-tools;
        };
      }
      (
        flake-utils.lib.eachDefaultSystem (
          system:
          let
            pkgs = nixpkgs.legacyPackages.${system};

            pkg = pkgs.buildGoModule {
              pname = "pia-tools";
              version = "1.3.0";
              src = ./.;
              vendorHash = "sha256-JG0kAdmlLv1aOa8y5S5IOKN8pWvqc6e8S8ApKGnA+G4=";
              meta = {
                description = "Toolset to manage wireguard tunnels to privateinternetaccess.com";
                homepage = "https://github.com/jdelkins/pia-tools";
                license = lib.licenses.mit;
              };
            };

            module = import ./module.nix { pia-tools = pkg; };
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

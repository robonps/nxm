{
  description = "Build nxm using buildGoModule";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        go = pkgs.go_1_24;
      in
      {
        packages = {
          default = pkgs.buildGoModule {
            pname = "nxm";
            version = "0.1";
            src = ./.;
            vendorHash = null;
            inherit go;
          };
        };

        devShell = pkgs.mkShell {
          buildInputs = [ go ];
          shellHook = ''
            echo "Go development shell for nxm"
          '';
        };
      }
    );
}
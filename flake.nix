{
  description = "Nix setup for ipk-omega-l4";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in {
        packages.default = pkgs.buildGoModule {
          pname = "ipk-omega-l4";
          version = "0.1.0";

          src = ./.;
          vendorHash = null;

          subPackages = [ "." ];

          ldflags = [ "-s" "-w" ];
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/main";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
          ];
        };
      });
}
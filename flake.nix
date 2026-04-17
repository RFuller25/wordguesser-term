{
  description = "WordGuesser - Terminal Wordle client";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          wordguesser = pkgs.buildGoModule {
            pname = "wordguesser";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-HsV9tFxW9vLAFHgVFrBopSqgdN/wAN1ss734rPQMbNM=";
            postInstall = ''
              mv $out/bin/wordguesser-term $out/bin/wordguesser
            '';
          };
          default = self.packages.${system}.wordguesser;
        };
      }
    );
}

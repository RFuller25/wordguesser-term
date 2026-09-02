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
            version = "0.3.0";
            src = ./.;
            vendorHash = "sha256-QsuRAXQXn6ko/SBAfAzntuMAlAsltCV8rBXAZL5Yxaw=";
            subPackages = [ "." ];
            postInstall = ''
              mv $out/bin/wordguesser-term $out/bin/wordguesser
            '';
          };
          quotes = pkgs.buildGoModule {
            pname = "quotes-term";
            version = "0.2.0";
            src = ./quotes-term;
            vendorHash = "sha256-HsV9tFxW9vLAFHgVFrBopSqgdN/wAN1ss734rPQMbNM=";
            postInstall = ''
              mv $out/bin/quotes-term $out/bin/quotes
            '';
          };
          default = self.packages.${system}.wordguesser;
        };
      }
    );
}

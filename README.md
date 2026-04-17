# WordGuesser

A terminal-based Wordle client built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Installation

### Option 1: Install directly (no config change required)

Run the following to install `wordguesser` into your user profile:

```bash
nix profile install path:/path/to/wordguesser-term
```

Replace `/path/to/wordguesser-term` with the actual path to this repository on your machine. The `wordguesser` binary will be available in your PATH immediately.

### Option 2: Add to your NixOS flake configuration

If you want `wordguesser` managed as part of your system configuration, convert your `/etc/nixos` setup to use flakes.

**1. Create `/etc/nixos/flake.nix`:**

```nix
{
  description = "NixOS configuration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    wordguesser.url = "path:/path/to/wordguesser-term";
  };

  outputs = { self, nixpkgs, wordguesser, ... }:
    let system = "x86_64-linux"; in {
      nixosConfigurations.nixos = nixpkgs.lib.nixosSystem {
        inherit system;
        modules = [
          ./configuration.nix
          {
            environment.systemPackages = [
              wordguesser.packages.${system}.default
            ];
          }
        ];
      };
    };
}
```

Replace `/path/to/wordguesser-term` with the actual path to this repository, and replace `nixos` in `nixosConfigurations.nixos` with your hostname if different.

**2. Rebuild your system:**

```bash
sudo nixos-rebuild switch --flake /etc/nixos#nixos
```

Again, replace `nixos` at the end with your hostname if needed.

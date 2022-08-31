{ pkgs ? import <nixpkgs> }:

with pkgs;
mkShell {
  buildInputs = [
    gnused
    go_1_19
    golangci-lint
    gopls
    nix-prefetch
    snitch
  ];
}

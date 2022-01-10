{ pkgs ? import <nixpkgs> }:

with pkgs;
mkShell {
  buildInputs = [
    gnused
    go_1_16
    golangci-lint
    gopls
    nix-prefetch
    snitch
  ];
}

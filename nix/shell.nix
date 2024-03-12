{ pkgs ? import <nixpkgs> }:

with pkgs;
mkShell {
  buildInputs = [
    gnused
    go_1_22
    golangci-lint
    gopls
    jsonnet-language-server
    nix-prefetch
    snitch
  ];
}

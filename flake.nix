{
  description = "jsonnet-language-server shell development tooling";

  inputs.nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlay =
        (
          final: prev: {
            jsonnet-language-server = prev.callPackage ./default.nix { pkgs = prev; };
            snitch = prev.callPackage ./snitch.nix { pkgs = prev; };
          }
        );
    } // (
      flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs { inherit system; overlays = [ self.overlay ]; };
        in
        {
          defaultPackage = pkgs.jsonnet-language-server;
          devShell = import ./shell.nix { inherit pkgs; };
          packages = {
            jsonnet-tool = pkgs.jsonnet-language-server;
          };
        }
      )
    );
}

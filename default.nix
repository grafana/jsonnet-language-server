{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.1.1";

  src = lib.cleanSource ./.;
  vendorSha256 = "sha256-jMA+lEzQ60p5JsFdPisIy0fK0f1N0RG9yfqpq4uOIn0=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/jdbaldry/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

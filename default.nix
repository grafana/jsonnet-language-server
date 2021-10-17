{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.0.1";

  meta = with lib; { maintainers = with maintainers; [ jdbaldry ]; };
  src = lib.cleanSource ./.;
  vendorSha256 = "sha256-jMA+lEzQ60p5JsFdPisIy0fK0f1N0RG9yfqpq4uOIn0=";
}

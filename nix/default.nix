{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.13.1";

  ldflags = ''
    -X main.version=${version}
  '';
  src = lib.cleanSource ../.;
  vendorHash = "sha256-+9Eh40kkyZc9mS4m6BqK5PweFRUA0iWgsG/h2jZJr5w=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/grafana/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry trevorwhitney ];
  };
}

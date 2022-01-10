{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.6.3";

  ldflags = ''
    -X main.version=${version}
  '';
  src = lib.cleanSource ../.;
  vendorSha256 = "sha256-mGocX5z3wf9KRhE9leLNCzn8sVdjKJo6FzgP1OwQB3M=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/grafana/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

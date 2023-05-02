{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.12.0";

  ldflags = ''
    -X main.version=${version}
  '';
  src = lib.cleanSource ../.;
  vendorSha256 = "sha256-lC3GAOJ/XVzn+9kk4PnW/7UwqjiXP7DqYmqauwOqQ+k=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/grafana/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

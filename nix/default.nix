{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.11.0";

  ldflags = ''
    -X main.version=${version}
  '';
  src = lib.cleanSource ../.;
  vendorSha256 = "sha256-ZyTo79M5nqtqrtTOGanzgHcnSvqCKACacNBWzhYG5nY=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/grafana/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

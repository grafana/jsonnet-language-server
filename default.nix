{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-language-server";
  version = "0.2.1";

  buildFlagsArray = ''
    -ldflags=
    -X main.version=${version}
  '';
  src = lib.cleanSource ./.;
  vendorSha256 = "sha256-8jX2we1fpVmjhwcaLZ584MdbkvnrcDNAw9xKhT/z740=";

  meta = with lib; {
    description = "A Language Server Protocol server for Jsonnet";
    homepage = "https://github.com/jdbaldry/jsonnet-language-server";
    license = licenses.agpl3;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "snitch";
  version = "7c727e0b7919ea504c07c8af0e65b48f07a9e87c";

  src = fetchFromGitHub {
    owner = "tsoding";
    repo = pname;
    rev = version;
    sha256 = "sha256-bflHSWN/BH4TSTTP4M3DldVwkV8MUSVCO15eYJTtTi0=";
  };
  vendorSha256 = "sha256-QAbxld0UY7jO9ommX7VrPKOWEiFPmD/xw02EZL6628A=";

  meta = with lib; {
    description = "Language agnostic tool that collects TODOs in the source code and reports them as Issues";
    homepage = "https://github.com/tsoding/snitch";
    license = licenses.mit;
    maintainers = with maintainers; [ jdbaldry ];
  };
}

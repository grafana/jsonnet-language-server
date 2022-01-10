# Releases

## Nix flake

A [Nix Flake](https://nixos.wiki/wiki/Flakes) is provided that can be installed using the Nix package manager.
On each release, the derivation `version` attribute needs to be updated with the release tag and the `vendorSha256` attribute with
the checksum for the fixed output derivation for the vendored Go packages.

> **Note:** The following steps require a 2.X release of the `nix` command.

1. Update the `version` with the release tag.
```patch
--- a/nix/default.nix
+++ b/nix/default.nix
@@ -3,7 +3,7 @@
 with pkgs;
 buildGoModule rec {
   pname = "jsonnet-language-server";
-  version = "0.6.3";
+  version = "<major>.<minor>.<patch>";

   ldflags = ''
     -X main.version=${version}
```
2. Replace the `vendorSha256` with `lib.fakeSha256`.
```patch
--- a/nix/default.nix
+++ b/nix/default.nix
@@ -9,7 +9,7 @@ buildGoModule rec {
     -X main.version=${version}
   '';
   src = lib.cleanSource ../.;
-  vendorSha256 = "sha256-mGocX5z3wf9KRhE9leLNCzn8sVdjKJo6FzgP1OwQB3M=";
+  vendorSha256 = lib.fakeSha256;
 
   meta = with lib; {
     description = "A Language Server Protocol server for Jsonnet";
```
3. Attempt to build the package using `nix build`.
This command will error but the output will provide expected checksum for the vendored packages.
```console
$ cd nix
$ nix build
warning: Git tree '/home/jdb/ext/grafana/jsonnet-language-server/jsonnet-language-server' is dirty
error: hash mismatch in fixed-output derivation '/nix/store/7p66cd269hnsgli1js7hb3lg2498kwfm-jsonnet-language-server-0.6.3-go-modules.drv':
         specified: sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
            got:    sha256-mGocX5z3wf9KRhE9leLNCzn8sVdjKJo6FzgP1OwQB3B=
error: 1 dependencies of derivation '/nix/store/90f116zvv0lx1p15kah9psrb162aarhh-jsonnet-language-server-0.6.3.drv' failed to build
```
4. Update the `vendorSha256` with the expected checksum from the previous commands output.
```patch
--- a/nix/default.nix
+++ b/nix/default.nix
@@ -9,7 +9,7 @@ buildGoModule rec {
     -X main.version=${version}
   '';
   src = lib.cleanSource ../.;
-  vendorSha256 = lib.fakeSha256;
+  vendorSha256 = "sha256-mGocX5z3wf9KRhE9leLNCzn8sVdjKJo6FzgP1OwQB3B=";
 
   meta = with lib; {
     description = "A Language Server Protocol server for Jsonnet";
```
5. Confirm the build succeeds with `nix build`.
```console
$ cd nix
$ nix build
$ ./result/bin/jsonnet-language-server --version
jsonnet-language-server version <major>.<minor>.<patch>
```
6. Commit and push the changes.

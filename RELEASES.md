# Releases

## Nix flake

A [Nix Flake](https://nixos.wiki/wiki/Flakes) is provided that can be installed using the Nix package manager.
On each release, the derivation `version` attribute needs to be updated with the release tag and the `vendorSha256` attribute with
the checksum for the fixed output derivation for the vendored Go packages.

> **Note:** The following steps require a 2.X release of the `nix` command.

```console
cd nix
nix develop
./release <major>.<minor>.<patch>
```

You can also do it with Docker:

```console
docker run -it -v /tmp:/tmp -v $(pwd):/workdir -w /workdir nixos/nix
cd nix
nix develop --extra-experimental-features nix-command --extra-experimental-features flakes
./release <major>.<minor>.<patch>
```

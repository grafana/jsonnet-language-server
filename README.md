# Jsonnet Language Server

A **[Language Server Protocol (LSP)](https://langserver.org)** server for [Jsonnet](https://jsonnet.org).

## Features

### Jump to definition

https://user-images.githubusercontent.com/29210090/145594957-efc01d97-d4c1-4fad-85cb-f5cb4a5f0e97.mp4

https://user-images.githubusercontent.com/29210090/145594976-670fff41-55e9-4ff9-b104-b5ac1cf77b42.mp4

### Error/Warning Diagnostics

https://user-images.githubusercontent.com/29210090/145595007-59dd4276-e8c2-451e-a1d9-bfc7fd83923f.mp4

### Linting Diagnostics

https://user-images.githubusercontent.com/29210090/145595044-ca3f09cf-5806-4586-8aa8-720b6927bc6d.mp4

### Standard Library Hover and Autocomplete

https://user-images.githubusercontent.com/29210090/145595059-e34c6d25-eff3-41df-ae4a-d3713ee35360.mp4

### Formatting

## Installation

To install the LSP server with Go into \"\${GOPATH}\"/bin:

```console
go get -u github.com/grafana/jsonnet-language-server
```

To download the latest release binary from GitHub:

``` {#Download from GitHub .shell}
curl -Lo jsonnet-language-server https://github.com/grafana/jsonnet-language-server/releases/latest/download/jsonnet-language-server
```

## Contributing

Contributions are more than welcome and I will try my best to be prompt
with reviews.

### Commits

Individual commits should be meaningful and have useful commit messages.
For tips on writing commit messages, refer to [How to write a commit
message](https://chris.beams.io/posts/git-commit/). Contributions will
be rebased before merge to ensure a fast-forward merge.

### [Developer Certificate of Origin (DCO)](https://github.com/probot/dco#how-it-works)

Contributors must sign the DCO for their contributions to be accepted.

### Code style

Go code should be formatted with `gofmt` and linted with
[golangci-lint](https://golangci-lint.run/).

## Editor integration

### Emacs

Refer to
[editor/jsonnet-language-server.el](editor/jsonnet-language-server.el)
for an example of enabling the LSP server with lsp-mode.

### [VSCodium](https://github.com/VSCodium/vscodium) / VSCode

Use the [vscode-jsonnet
extension](https://github.com/julienduchesne/vscode-jsonnet)

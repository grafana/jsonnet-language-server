# Jsonnet Language Server

A **[Language Server Protocol (LSP)](https://langserver.org)** server for [Jsonnet](https://jsonnet.org).

## Features

### Jump to definition

https://user-images.githubusercontent.com/29210090/145594957-efc01d97-d4c1-4fad-85cb-f5cb4a5f0e97.mp4

https://user-images.githubusercontent.com/29210090/145594976-670fff41-55e9-4ff9-b104-b5ac1cf77b42.mp4

https://user-images.githubusercontent.com/29210090/154743159-81adf3b3-e929-4731-8b23-718085d222c5.mp4

### Error/Warning Diagnostics

https://user-images.githubusercontent.com/29210090/145595007-59dd4276-e8c2-451e-a1d9-bfc7fd83923f.mp4

### Linting Diagnostics

https://user-images.githubusercontent.com/29210090/145595044-ca3f09cf-5806-4586-8aa8-720b6927bc6d.mp4

### Standard Library Hover and Autocomplete

https://user-images.githubusercontent.com/29210090/145595059-e34c6d25-eff3-41df-ae4a-d3713ee35360.mp4

### Formatting

## Installation

Download the latest release binary from GitHub: https://github.com/grafana/jsonnet-language-server/releases

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

* Emacs: Refer to [editor/jsonnet-language-server.el](editor/jsonnet-language-server.el)
* Vim: Refer to [editor/jsonnet-language-server.vim](editor/jsonnet-language-server.vim)
* VSCod(e|ium): Use the [Jsonnet Language Server extension](https://marketplace.visualstudio.com/items?itemName=Grafana.vscode-jsonnet) ([source code](https://github.com/grafana/vscode-jsonnet))
* Jetbrains: Use the [Jsonnet Language Server plugin](https://plugins.jetbrains.com/plugin/18752-jsonnet-language-server) ([source code](https://github.com/zzehring/intellij-jsonnet))

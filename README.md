# Jsonnet Language Server

A **[Language Server Protocol (LSP)](https://langserver.org)** server for [Jsonnet](https://jsonnet.org).

## Features

### Jump to definition

![self Support Demo](./examples/self-support.mp4)
![dollar Support Demo](./examples/dollar-support.mp4)

### Error/Warning Diagnostics

![Error Support Demo](./examples/error-support.mp4)

### Linting Diagnostics

![Linting Support Demo](./examples/linting-support.mp4)

### Standard Library Hover and Autocomplete

![stdlib Support Demo](./examples/stdlib-support.mp4)

### Formatting

TODO

## Installation

To install the LSP server with Go into \"\${GOPATH}\"/bin:

```console
go get -u github.com/jdbaldry/jsonnet-language-server
```

To download the latest release binary from GitHub:

``` {#Download from GitHub .shell}
curl -Lo jsonnet-language-server https://github.com/jdbaldry/jsonnet-language-server/releases/latest/download/jsonnet-language-server
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

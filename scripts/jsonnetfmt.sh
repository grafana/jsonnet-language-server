#!/usr/bin/env bash

set -euxo pipefail

# Format all jsonnet files in the repo, with exceptions.
# Ignore vendor/ dir

exceptions=("./pkg/server/testdata/hover-error.jsonnet")

for f in $(find . -name '*.jsonnet' -print -o -name '*.libsonnet' -not -path "*/vendor/*" -print); do
    if [[ " ${exceptions[@]} " =~ " ${f} " ]]; then
        continue
    fi
    jsonnetfmt -i "$f"
done

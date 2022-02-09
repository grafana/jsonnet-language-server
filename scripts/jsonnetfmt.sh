#!/usr/bin/env bash

for f in $(find . -name '*.jsonnet' -print -o -name '*.libsonnet' -print); do
    jsonnetfmt -i "$f" || echo "Error formatting $f. May be expected (some tests include invalid jsonnet)."
done
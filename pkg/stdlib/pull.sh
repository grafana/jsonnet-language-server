#!/usr/bin/env bash

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

curl https://raw.githubusercontent.com/google/jsonnet/master/doc/_stdlib_gen/html.libsonnet -o "${SCRIPT_DIR}/html.libsonnet"
curl https://raw.githubusercontent.com/google/jsonnet/master/doc/_stdlib_gen/stdlib-content.jsonnet -o "${SCRIPT_DIR}/stdlib-content.jsonnet"
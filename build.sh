#!/bin/bash

set -euxo pipefail
go build ./...
go build ./cli/lbryschema-cli.go
go build -o lbryschema-python-binding.so -buildmode=c-shared ./binding/lbryschema-python-binding.go

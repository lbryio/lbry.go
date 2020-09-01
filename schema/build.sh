#!/bin/bash

set -euxo pipefail
go build ./...
go build ./cli/lbryschema-cli.go

BINARY=lbry

DIR = $(shell cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)

VERSION=$(shell git --git-dir=${DIR}/.git describe --dirty --always --long --abbrev=7)
LDFLAGS = -ldflags "-X main.Version=${VERSION}"


.PHONY: build clean
.DEFAULT_GOAL: build


build:
	CGO_ENABLED=0 go build ${LDFLAGS} -asmflags -trimpath=${DIR} -o ${DIR}/${BINARY} main.go

clean:
	if [ -f ${DIR}/${BINARY} ]; then rm ${DIR}/${BINARY}; fi

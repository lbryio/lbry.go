BINARY=lbry

DIR = $(shell cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)

VERSION=$(shell git --git-dir=${DIR}/.git describe --dirty --always)
COMMIT=$(shell git --git-dir=${DIR}/.git rev-parse --short HEAD)
BRANCH=$(shell git --git-dir=${DIR}/.git rev-parse --abbrev-ref HEAD)
LDFLAGS = -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH}"


.PHONY: build dep clean
.DEFAULT_GOAL: build


build:
	CGO_ENABLED=0 go build ${LDFLAGS} -asmflags -trimpath -o ${DIR}/${BINARY} main.go

#dep:
#	go get github.com/golang/dep/cmd/dep && dep ensure

clean:
	if [ -f ${DIR}/${BINARY} ]; then rm ${DIR}/${BINARY}; fi

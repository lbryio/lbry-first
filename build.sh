#!/bin/bash

 set -euo pipefail

 DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
 cd "$DIR"
 DIR="$PWD"


echo "== Installing dependencies =="
GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports
GO111MODULE=off go get -u github.com/jteeuwen/go-bindata/...
go mod download


echo "== Checking dependencies =="
go mod verify
set -e


echo "== Compiling =="
export IMPORTPATH="github.com/lbryio/lbry-first"
mkdir -p "$DIR/bin"
go generate -v
export VERSIONSHORT="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty)"}"
export VERSIONLONG="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty --long)"}"
export COMMITMSG="$(echo ${TRAVIS_COMMIT_MESSAGE:-"$(git show -s --format=%s)"} | tr -d '"' | head -n 1)"
touch -a .env && set -o allexport
source ./.env; set +o allexport
CGO_ENABLED=0 GOARCH=amd64 go build -v -o "./bin/lbry-first" -asmflags -trimpath="$DIR" -ldflags "-X ${IMPORTPATH}/commands/server/services/youtube.clientSecret=${CLIENTSECRET} -X ${IMPORTPATH}/meta.version=${VERSIONSHORT} -X ${IMPORTPATH}/meta.versionLong=${VERSIONLONG} -X \"${IMPORTPATH}/meta.commitMsg=${COMMITMSG}\""
echo "== Done building version $("$DIR/bin/lbry-first" version) =="
#CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -o "./bin/lbry-first-darwin" -asmflags -trimpath="$DIR" -ldflags "-X ${IMPORTPATH}/commands/server/services/youtube.clientSecret=${CLIENTSECRET} -X ${IMPORTPATH}/meta.version=${VERSIONSHORT} -X ${IMPORTPATH}/meta.versionLong=${VERSIONLONG} -X \"${IMPORTPATH}/meta.commitMsg=${COMMITMSG}\""
#echo "== Done building darwin version $("$DIR/bin/lbry-first-darwin" version) =="
#CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -o "./bin/lbry-first.exe" -asmflags -trimpath="$DIR" -ldflags "-X ${IMPORTPATH}/commands/server/services/youtube.clientSecret=${CLIENTSECRET} -X ${IMPORTPATH}/meta.version=${VERSIONSHORT} -X ${IMPORTPATH}/meta.versionLong=${VERSIONLONG} -X \"${IMPORTPATH}/meta.commitMsg=${COMMITMSG}\""
#echo "== Done building windows version $("$DIR/bin/lbry-first.exe" version) =="

echo "$(git describe --tags --always --dirty)" > ./bin/lbry-first.txt
chmod +x ./bin/lbry-first
exit 0
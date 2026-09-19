#!/bin/bash -x
APP_NAME=keyring-winbridge.exe
BUILD_OPTIONS="-w -s"
#: windows/amd64 with the broadest amd64 baseline is enforced as a target
export GOOS=windows
export GOARCH=amd64
export GOAMD64=v1

if [[ ! -d bin ]]; then
    mkdir bin
fi

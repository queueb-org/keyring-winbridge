#!/bin/bash -e

functions="$(dirname "$0")/.build-env.sh"
if [ -f "$functions" ]; then
  # shellcheck disable=SC1090
  source "$functions"
fi

go test \
  -count=1 \
  -run '^TestWindowsCredentialCRUD$' \
  -tags integration \
  ./internal/credential

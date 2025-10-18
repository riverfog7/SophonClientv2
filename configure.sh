#!/bin/bash

base64 -d -i internal/secrets/secrets.b64 > internal/secrets/secrets.go

pushd proto
protoc --go_out=../internal/models --go_opt=paths=source_relative manifest.proto
protoc --go_out=../internal/models --go_opt=paths=source_relative manifest_ldiff.proto
popd

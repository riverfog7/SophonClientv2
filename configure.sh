#!/bin/bash

pushd proto
protoc --go_out=../internal/models --go_opt=paths=source_relative manifest.proto
protoc --go_out=../internal/models --go_opt=paths=source_relative manifest_ldiff.proto
popd

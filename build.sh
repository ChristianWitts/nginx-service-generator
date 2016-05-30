#!/bin/bash

__version__="0.1"
release_root="bin/releases/${__version__}"

mkdir -p ${release_root}/{linux_64,osx}

# Build for linux
GOOS=linux
GOARCH=amd64
go build -o "${release_root}/linux_64/service-generator"

# Build for darwin
GOOS=darwin
GOARCH=amd64
go build -o "${release_root}/osx/service-generator"

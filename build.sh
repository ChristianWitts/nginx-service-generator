#!/bin/bash

function consoleLog {
    echo '['$(date +'%a %Y-%m-%d %H:%M:%S %z')']' "$1"
}

read -r -d '' usage <<-'EOF' || true
build.sh --version 0.2.0 --release

    -v --version    [arg] The version you're building
    -r --release          If you're tagging this as a release build
    -h --help             This page
EOF

function help {
    echo "" 1>&2
    echo " ${usage}" 1>&2
    echo "" 1>&2
    exit 1
}

function build {
    consoleLog "Building for version ${1}"
    BUILD_ROOT="bin/releases/${1}"
    mkdir -p ${BUILD_ROOT}/{linux_64,darwin}

    consoleLog "Building Linux amd64 binary"
    GOOS=linux; GOARCH=amd64 go build \
        -ldflags="-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD` -X main.version=${1}" \
        -o ${BUILD_ROOT}/linux_64/service-generator

    consoleLog "Building Darwin amd64 binary"
    GOOS=darwin; GOARCH=amd64 go build \
        -ldflags="-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD` -X main.version=${1}" \
        -o ${BUILD_ROOT}/darwin/service-generator
}

function buildRelease {
    consoleLog "Generating releases for version ${1}"
    RELEASE_ROOT="bin/releases/${1}"
    (cd $RELEASE_ROOT; tar -czvf "service-generator_darwin-${1}.tar.gz" -C darwin/ service-generator)
    (cd $RELEASE_ROOT; tar -czvf "service-generator_linux64-${1}.tar.gz" -C linux_64/ service-generator)
}

VERSION="-1"
RELEASE="N"

while :
do
    case "$1" in
        -v | --version)
            VERSION="$2"
            shift 2
            ;;
        -r | --release)
            RELEASE="Y"
            shift 1
            ;;
        -h | --help)
            help
            ;;
        --)
            shift
            break
            ;;
        -*)
            echo "Error: Unknown option: $1" >$2
            exit 1
            ;;
        *)
            break
            ;;
    esac
done

if [ "$VERSION" = "-1" ]; then
    help
fi

build $VERSION

if [ "$RELEASE" = "Y" ]; then
    buildRelease $VERSION
fi

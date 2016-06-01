#!/bin/bash

# Supported targets
TARGETS=(darwin dragonfly freebsd linux netbsd openbsd plan9 solaris windows)

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
    export GOARCH=amd64

    for TARGET in ${TARGETS[@]};
    do
        local TARGET_DIR=${BUILD_ROOT}/${TARGET}_amd64
        mkdir -p $TARGET_DIR
        consoleLog "Building $TARGET amd64 binary"
        export GOOS=$TARGET
        go build \
            -ldflags="-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD` -X main.version=${1}" \
            -o ${TARGET_DIR}/service-generator
    done
}

function buildRelease {
    consoleLog "Generating releases for version ${1}"
    local RELEASE_ROOT="bin/releases/${1}"
    for TARGET in ${TARGETS[@]};
    do
        (cd $RELEASE_ROOT; tar -czf "service-generator_${TARGET}_amd64-${1}.tar.gz" -C ${TARGET}_amd64/ service-generator)
        rm -rf ${RELEASE_ROOT}/${TARGET}_amd64/
    done
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

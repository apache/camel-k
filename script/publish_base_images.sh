#!/bin/sh

set -e

location=$(dirname $0)
rootdir=$(realpath $location/../)

if [ "$#" -ne 1 ]; then
    echo "usage: $0 runtime-version"
    exit 1
fi

echo "Start publishing base images with runtime $1"

$rootdir/publisher --runtime-version=$1 ""

echo "All base images have been published"

#!/bin/sh

set -e

location=$(dirname $0)
rootdir=$(realpath $location/../)

echo "Start publishing base images"

$rootdir/publisher

echo "All base images have been published"

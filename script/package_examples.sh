#!/bin/sh

location=$(dirname $0)
rootdir=$(realpath $location/../)

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

version=$1

tar -zcvf $rootdir/camel-k-examples-$version.tar.gz examples

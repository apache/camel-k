#!/bin/sh

location=$(dirname $0)
rootdir=$(realpath $location/../)

version=$($location/get_version.sh)

tar -zcvf $rootdir/camel-k-examples-$version.tar.gz examples

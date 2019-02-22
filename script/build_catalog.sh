#!/bin/sh

location=$(dirname $0)
rootdir=$location/../

if [ "$#" -ne 2 ]; then
    echo "usage: $0 catalog.version runtime.version"
    exit 1
fi

$rootdir/mvnw -q \
    -f ${rootdir}/build/maven/pom-catalog.xml \
    -Dcatalog.path=${rootdir}/deploy \
    -Dcatalog.version=$1 \
    -Druntime.version=$2
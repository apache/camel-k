#!/bin/sh

set -e

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

location=$(dirname $0)
version=$1

for f in $(find $location/../deploy -type f -name "*.yaml"); 
do
    sed -i -r "s/docker.io\/apache\/camel-k:([0-9]+[a-zA-Z0-9\-\.].*).*/docker.io\/apache\/camel-k:${version}/" $f
done

echo "Camel K version set to: $version"

#!/bin/sh

set -e

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "usage: $0 version [image_name]"
    exit 1
fi

location=$(dirname $0)
version=$1
image_name=${2:-docker.io\/apache\/camel-k}
sanitized_image_name=${image_name//\//\\\/}


for f in $(find $location/../deploy -type f -name "*.yaml"); 
do
    sed -i -r "s/docker.io\/apache\/camel-k:([0-9]+[a-zA-Z0-9\-\.].*).*/${sanitized_image_name}:${version}/" $f
done

echo "Camel K version set to: $version and image name to: $image_name"

#!/bin/sh

location=$(dirname $0)
rootdir=$location/../

version=$($location/get_version.sh)

./mvnw -f runtime/pom.xml \
    -N \
    -Pcatalog \
    -Dcatalog.path=${rootdir}/deploy/camel-catalog.yaml

#!/bin/sh

location=$(dirname $0)
rootdir=$location/../

version=$($location/get_version.sh)

./mvnw -f runtime/pom.xml \
    -N \
    -D catalog.path=${rootdir}/deploy/camel-catalog.yaml \
    org.apache.camel.k:camel-k-maven-plugin:${version}:generate-catalog

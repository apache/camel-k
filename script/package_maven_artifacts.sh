#!/bin/sh

location=$(dirname $0)

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

cd ${location}/..

./mvnw \
    -f build/maven/pom-runtime.xml \
    -DoutputDirectory=$PWD/build/_maven_output \
    -Druntime.version=$1 \
    dependency:copy-dependencies
#!/bin/sh

location=$(dirname $0)
rootdir=$location/../

version=$($location/get_version.sh)

if [[ "$#" -eq 0 ]]; then
    ./mvnw -f runtime/pom.xml \
        -N \
        -Pcatalog \
        -Dcatalog.path=${rootdir}/deploy
else
    for ver in "$@"
    do
        ./mvnw -f runtime/pom.xml \
            -N \
            -Pcatalog \
            -Dcatalog.version=$ver \
            -Dcatalog.path=${rootdir}/deploy
    done
fi

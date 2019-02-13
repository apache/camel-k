#!/bin/sh

location=$(dirname $0)
rootdir=$location/../

version=$($location/get_version.sh)

if [ "$#" -eq 0 ]; then
    echo "build default catalog"
    ./mvnw -q -f runtime/pom.xml \
        -N \
        -Pcatalog \
        -Dcatalog.path=${rootdir}/deploy
else
    for ver in "$@"
    do
        echo "build catalog for version $ver"
        ./mvnw -q -f runtime/pom.xml \
            -N \
            -Pcatalog \
            -Dcatalog.version=$ver \
            -Dcatalog.path=${rootdir}/deploy
    done
fi

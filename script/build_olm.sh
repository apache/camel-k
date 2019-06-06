#!/bin/sh

location=$(dirname $0)
olm_catalog=${location}/../deploy/olm-catalog

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

version=$1

cd $location/..

operator-sdk olm-catalog gen-csv --csv-version ${version} --csv-config deploy/olm-catalog/csv-config.yaml --update-crds

rm $olm_catalog/camel-k/${version}/crd-*.yaml
cp $location/../deploy/crd-*.yaml $olm_catalog/camel-k/${version}/

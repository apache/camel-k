#!/bin/sh

set -e

location=$(dirname $0)

version=$($location/get_version.sh)
version_num=$(echo $version | sed -E "s/([0-9.]*)-SNAPSHOT/\1/g")

$location/set_version.sh $version_num

echo "Camel K prepared for releasing version: $version_num"

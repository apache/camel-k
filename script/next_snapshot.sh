#!/bin/sh

set -e

location=$(dirname $0)
global_version_file=$location/../version/version.go

version=$($location/get_version.sh)
version_num=$(echo $version | sed -E "s/([0-9.]*)-SNAPSHOT/\1/g")
next_version_num=$(echo $version_num | awk 'BEGIN { FS = "." } ; {print $1"."$2"."++$3}')
next_version="$next_version_num-SNAPSHOT"

echo "Increasing version to $next_version"

$location/set_version.sh $next_version

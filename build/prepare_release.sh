#!/bin/sh

set -e

location=$(dirname $0)
global_version_file=$location/../version/version.go

# Set the new global version by removing "-SNAPSHOT"
sed -i "s/-SNAPSHOT//g" $global_version_file

# Get the new version
version=$($location/get_version.sh)

# Updating the Java modules
mvn versions:set -DgenerateBackupPoms=false -DnewVersion=$version -f $location/../runtime

echo "Camel K prepared for releasing version: $version"


#!/bin/sh

set -e

location=$(dirname $0)
new_version=$1

global_version_file=$location/../version/version.go
version=$($location/get_version.sh)

# Set the new global version
sed -i "s/$version/$new_version/g" $global_version_file
find $location/../deploy -type f -exec sed -i "s/$version/$new_version/g" {} \;

# Updating the Java modules
./mvnw versions:set -DgenerateBackupPoms=false -DnewVersion=$new_version -f $location/../runtime

echo "Camel K version set to: $new_version"


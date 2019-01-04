#!/bin/sh

set -e

location=$(dirname $0)
version=$($location/get_version.sh)

git branch -D staging-$version || true
git checkout -b staging-$version
git add * || true
git commit -a -m "Release $version"
git tag --force $version staging-$version
git push --force upstream $version

echo "Tag $version pushed upstream"

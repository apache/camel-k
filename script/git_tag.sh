#!/bin/sh

set -e

if [ "$#" -ne 2 ]; then
    echo "usage: $0 version branch"
    exit 1
fi

location=$(dirname $0)
target_version=$1
target_branch=$2

git branch -D staging-${target_version} || true
git checkout -b staging-${target_version}
git add * || true
git commit -a -m "Release ${target_version}"
git tag --force ${target_version} staging-${target_version}
git push --force ${target_branch} ${target_version}

echo "Tag ${target_version} pushed ${target_branch}"

#!/bin/bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <Camel K version>"
  exit 1
fi

location=$(dirname $0)
version="$1"

re="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)$"
if ! [[ $version =~ $re ]]; then
    echo "❗ argument must match semantic version: $version"
    exit 1
fi
version_mm="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"

new_release="v$(echo "$version_mm" | tr \. _)_x"
new_release_branch="release-$version_mm.x"

# pick the oldest release (we will replace it)
oldest_release=$(yq '.jobs[] | key | select ( . !="main" )' $location/../.github/workflows/release.yml | sort | head -1)
oldest_release_branch=$(yq ".jobs[\"$oldest_release\"].steps[0].with.branch-ref" $location/../.github/workflows/release.yml)
echo "Swapping GH actions tasks from $oldest_release to $new_release, $oldest_release_branch to $new_release_branch"
# Nightly release action
sed -i "s/$oldest_release/$new_release/g" $location/../.github/workflows/release.yml
sed -i "s/$oldest_release_branch/$new_release_branch/g" $location/../.github/workflows/release.yml
# Automatic updates action
sed -i "s/$oldest_release/$new_release/g" $location/../.github/workflows/automatic-updates.yml
sed -i "s/$oldest_release_branch/$new_release_branch/g" $location/../.github/workflows/automatic-updates.yml
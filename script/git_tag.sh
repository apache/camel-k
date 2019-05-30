#!/bin/sh

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

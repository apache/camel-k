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
    echo "usage: $0 version remote"
    exit 1
fi

location=$(dirname $0)
target_version=$1
target_tag=v$target_version
target_staging=staging-$target_tag
target_remote=$2

git branch -D ${target_staging} || true
git checkout -b ${target_staging}
git add * || true
git commit -a -m "Release ${target_version}" || true

git tag --force ${target_tag} ${target_staging}
git push --force ${target_remote} ${target_tag}
echo "Tag ${target_tag} pushed to ${target_remote}"

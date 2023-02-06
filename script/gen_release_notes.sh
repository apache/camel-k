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

if [ "$#" -ne 3 ]; then
    echo "usage: $0 last-version new-version branch"
    exit 1
fi

location=$(dirname $0)
last_tag=v$1
new_tag=v$2
branch=$3

echo "Generating release notes for version $new_tag starting from tag $last_tag"

start_sha=$(git rev-list -n 1 $last_tag)
if [ "$start_sha" == "" ]; then
	echo "cannot determine initial SHA from tag $last_tag"
	exit 1
fi
echo "Using start SHA $start_sha from tag $last_tag"

set +e
end_sha=$(git rev-list -n 1 $new_tag 2>&1)
if [ $? -ne 0 ]; then
	end_sha=$(git rev-parse upstream/$branch)
    if [ "$end_sha" == "" ]; then
    	echo "cannot determine current SHA from git"
    	exit 1
    fi
    echo "Using end SHA $end_sha from upstream/$branch"
else
	echo "Using end SHA $end_sha from tag $new_tag"
fi
set -e

set +e
which release-notes > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "No \"release-notes\" command found. Please install it from https://github.com/kubernetes/release"
  exit 1
fi
set -e

set +e
if [ -z "${GITHUB_TOKEN}" ]; then
  echo "No \"GITHUB_TOKEN\" environment variable exported. Please set the GITHUB_TOKEN environment variable"
  exit 1
fi
set -e

release-notes --start-sha $start_sha --end-sha $end_sha --branch $branch --repo camel-k --org apache --output $location/../release-notes.md --required-author ""

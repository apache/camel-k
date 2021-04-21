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
    echo "usage: $0 last-tag new-tag"
    exit 1
fi

location=$(dirname $0)
last_tag=v$1
new_tag=v$2

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
	end_sha=$(git rev-parse upstream/main)
    if [ "$end_sha" == "" ]; then
    	echo "cannot determine current SHA from git"
    	exit 1
    fi
    echo "Using end SHA $end_sha from upstream/main"
else
	echo "Using end SHA $end_sha from tag $new_tag"
fi
set -e

set +e
which release-notes > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "No \"release-notes\" command found. Please follow these steps to install it:"
  echo "  1) git clone git@github.com:nicolaferraro/release.git"
  echo "  2) cd release && go install ./cmd/release-notes/"
  echo ""
  exit 1
fi
set -e

cli_version=$(release-notes --version)
if [ "$cli_version" != "nicolaferraro" ]; then
  echo "You must install a specific fork from nicolaferraro of the \"release-notes\" command"
  exit 1
fi

release-notes --start-sha $start_sha --end-sha $end_sha --github-repo camel-k --github-org apache --release-version $new_tag --output $location/../release-notes.md --requiredAuthor ""

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

location=$(dirname $0)
rootdir=$location/../

cd $rootdir

#
# Requires directory to find the files in
#

dir=$1

if [ ! -d "$dir" ]; then
  echo "Error: Cannot find directory."
	exit 1
fi

created=$(date -u +%FT%TZ)

set +e
for file in `find "$dir" -type f`; do
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sed -i "s/createdAt: .*/createdAt: ${created}/" "${file}"
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    sed -i '' "s/createdAt: .*/createdAt: ${created}/" "${file}"
  fi
done
set -e

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
go build ./cmd/util/license-check/

#
# Requires directory to find the files in
# The header file containing the header text
#

dir=$1
header=$2
ignore=$3

if [ ! -d "$dir" ]; then
  echo "Error: Cannot find directory."
	exit 1
fi

if [ ! -f "$header" ]; then
	echo "Error: Cannot find header file."
	exit 1
fi

set +e
failed=0
find "$dir" -type f -print0 | while IFS= read -r -d '' file; do
  if [ -n "${ignore}" ] && [[ "${file}" == *"${ignore}"* ]]; then
    continue
  fi

  ./license-check "$file" "$header" &> /dev/null
	if [ $? -ne 0 ]; then
		cat "$header" <(echo) "$file" > "${file}.new"
		if [ $? -eq 0 ]; then
			mv "${file}.new" "${file}"
		fi
	fi
done
set -e

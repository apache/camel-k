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

location=$(dirname $0)
global_version_file=$location/../version/version.go

version=$(cd $location && make get-version)
version_num=$(echo $version | sed -E "s/([0-9.]*)-SNAPSHOT/\1/g")
next_version_num=$(echo $version_num | awk 'BEGIN { FS = "." } ; {print $1"."$2"."++$3}')
next_version="$next_version_num-SNAPSHOT"

echo "Increasing version to $next_version"

$location/set_version.sh $next_version

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

set -e

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

location=$(dirname $0)
target_version=$1
target_tag=v$target_version

api_rule="s/github.com\/apache\/camel-k\/pkg\/apis\/camel [A-Za-z0-9\.\-]+.*$/github.com\/apache\/camel-k\/pkg\/apis\/camel $target_tag/"
client_rule="s/github.com\/apache\/camel-k\/pkg\/client\/camel [A-Za-z0-9\.\-]+.*$/github.com\/apache\/camel-k\/pkg\/client\/camel $target_tag/"
kr_rule="s/github.com\/apache\/camel-k\/pkg\/kamelet\/repository [A-Za-z0-9\.\-]+.*$/github.com\/apache\/camel-k\/pkg\/kamelet\/repository $target_tag/"

sed -i -r "$api_rule"    $location/../go.mod
sed -i -r "$client_rule" $location/../go.mod
sed -i -r "$kr_rule"     $location/../go.mod
sed -i -r "$api_rule"    $location/../pkg/client/camel/go.mod
sed -i -r "$api_rule"    $location/../pkg/kamelet/repository/go.mod
sed -i -r "$client_rule" $location/../pkg/kamelet/repository/go.mod

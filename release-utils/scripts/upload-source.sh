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

if [ "$#" -lt 2 ]; then
    echo "usage: $0 upload-sources release-version destination-version"
    exit 1
fi

location=$(dirname $0)
version=$1

mkdir $1/
cd $1/

wget https://github.com/apache/camel-k/archive/refs/tags/v$1.tar.gz -O camel-k-sources-$1.tar.gz
cp ../../../camel-k-client-$1-* .
cp ../../../camel-k-examples-$1* .
cd ../
./sign.sh $1/
svn import $1/ https://dist.apache.org/repos/dist/dev/camel/camel-k/$2/ -m "Import camel-k release"

rm -rf $1/

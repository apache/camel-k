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

location=$(dirname $0)

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "usage: $0 version strategy [staging.repo]"
    exit 1
fi

version=$1
strategy=$2
staging_repo=$3

cd ${location}/..

if [ "$strategy" = "copy" ]; then
    mvn \
        -V \
        --no-transfer-progress \
        -f build/maven/pom-runtime.xml \
        -DoutputDirectory=$PWD/build/_maven_output \
        -Druntime.version=$version \
        -Dstaging.repo=$staging_repo \
        -Dalpn.jdk8.version="8.1.13.v20181017" \
        dependency:copy-dependencies
elif [ "$strategy" = "download" ]; then
    mvn \
        -V \
        --no-transfer-progress \
        -f build/maven/pom-runtime.xml \
        -Dmaven.repo.local=$PWD/build/_maven_output \
        -Druntime.version=$version \
        -Dstaging.repo=$staging_repo \
        -Dalpn.jdk8.version="8.1.13.v20181017" \
        install
else
    echo "unknown strategy: $strategy"
    exit 1
fi


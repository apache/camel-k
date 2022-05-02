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

if [ "$#" -lt 1 ]; then
    echo "usage: $0 <Camel K runtime version> [<local Camel K runtime project directory>]"
    exit 1
fi

camel_k_destination="$PWD/build/_maven_output"
camel_k_runtime_version=$1

if [ -z "$2" ]; then
    is_snapshot=$(echo "$1" | grep "SNAPSHOT")
    if [ "$is_snapshot" = "$1" ]; then
        echo "You're trying to package SNAPSHOT artifacts. You probably wants them from your local environment, try calling:"
        echo "$0 <Camel K runtime version> <local Camel K runtime project directory>"
        exit 3
    fi

    # Take the dependencies officially released
    wget https://repo1.maven.org/maven2/org/apache/camel/k/apache-camel-k-runtime/$1/apache-camel-k-runtime-$1-source-release.zip -O $PWD/build/apache-camel-k-runtime-$1-source-release.zip
    unzip -q -o $PWD/build/apache-camel-k-runtime-$1-source-release.zip -d $PWD/build
    mvn -q -f $PWD/build/apache-camel-k-runtime-$1/pom.xml \
        dependency:copy-dependencies \
        -DincludeScope=runtime \
        -Dmdep.copyPom=true \
        -DoutputDirectory=$camel_k_destination \
        -Dmdep.useRepositoryLayout=true
    rm -rf $PWD/build/apache-camel-k-runtime*
else
    # Take the dependencies from a local development environment
    camel_k_runtime_source=$2
    camel_k_runtime_source_version=$(mvn -f $camel_k_runtime_source/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout)

    if [ "$camel_k_runtime_version" != "$camel_k_runtime_source_version" ]; then
        echo "Misaligned version. You're building Camel K project using $camel_k_runtime_version but trying to bundle dependencies from local Camel K runtime $camel_k_runtime_source_version"
        exit 2
    fi

    mvn -q -f $camel_k_runtime_source/pom.xml \
    dependency:copy-dependencies \
        -DincludeScope=runtime \
        -Dmdep.copyPom=true \
        -DoutputDirectory=$camel_k_destination \
        -Dmdep.useRepositoryLayout=true
fi

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
rootdir=$(realpath ${location}/../)

while getopts "d:s:" opt; do
  case "${opt}" in
    d)
      local_runtime_dir="${OPTARG}"
      ;;
    s)
      staging_repo="${OPTARG}"
      ;;
    *)
      ;;
  esac
done
shift $((OPTIND-1))

if [ "$#" -lt 1 ]; then
    echo "usage: $0 [-s <staging repository>] [-d <local Camel K runtime project directory>] <Camel K runtime version>"
    exit 1
fi

options=""
if [ "$CI" = "true" ]; then
  options="--batch-mode"
fi

camel_k_destination="$PWD/build/_maven_output"
camel_k_runtime_version=$1
maven_repo=${staging_repo:-https://repo1.maven.org/maven2}

if [ -n "$staging_repo" ]; then
    if  [[ $staging_repo == http:* ]] || [[ $staging_repo == https:* ]]; then
        options="${options} -s ${rootdir}/settings.xml"
        cat << EOF > ${rootdir}/settings.xml
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <profiles>
    <profile>
      <id>camel-k-staging</id>
      <repositories>
        <repository>
          <id>camel-k-staging-releases</id>
          <name>Camel K Staging</name>
          <url>${staging_repo}</url>
          <releases>
            <enabled>true</enabled>
            <updatePolicy>never</updatePolicy>
          </releases>
          <snapshots>
            <enabled>false</enabled>
          </snapshots>
        </repository>
      </repositories>
    </profile>
  </profiles>
  <activeProfiles>
    <activeProfile>camel-k-staging</activeProfile>
  </activeProfiles>
</settings>
EOF
    fi
fi

if [ -z "${local_runtime_dir}" ]; then
    is_snapshot=$(echo "${camel_k_runtime_version}" | grep "SNAPSHOT")
    if [ "$is_snapshot" = "${camel_k_runtime_version}" ]; then
        echo "You're trying to package SNAPSHOT artifacts. You probably wants them from your local environment, try calling:"
        echo "$0 <Camel K runtime version> <local Camel K runtime project directory>"
        exit 3
    fi

    # Take the dependencies officially released
    wget ${maven_repo}/org/apache/camel/k/apache-camel-k-runtime/${camel_k_runtime_version}/apache-camel-k-runtime-${camel_k_runtime_version}-source-release.zip -O $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}-source-release.zip
    unzip -q -o $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}-source-release.zip -d $PWD/build
    mvn -q \
        $options \
        -f $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}/pom.xml \
        dependency:copy-dependencies \
        -DincludeScope=runtime \
        -Dmdep.copyPom=true \
        -DoutputDirectory=$camel_k_destination \
        -Dmdep.useRepositoryLayout=true
    rm -rf $PWD/build/apache-camel-k-runtime*
else
    # Take the dependencies from a local development environment
    camel_k_runtime_source=${local_runtime_dir}
    camel_k_runtime_source_version=$(mvn $options -f $camel_k_runtime_source/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout)

    if [ "$camel_k_runtime_version" != "$camel_k_runtime_source_version" ]; then
        echo "Misaligned version. You're building Camel K project using $camel_k_runtime_version but trying to bundle dependencies from local Camel K runtime $camel_k_runtime_source_version"
        exit 2
    fi

    mvn -q \
        $options \
        -f $camel_k_runtime_source/pom.xml \
    dependency:copy-dependencies \
        -DincludeScope=runtime \
        -Dmdep.copyPom=true \
        -DoutputDirectory=$camel_k_destination \
        -Dmdep.useRepositoryLayout=true
fi

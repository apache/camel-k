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

camel_k_destination="$PWD/build/_maven_output"
camel_k_runtime_version=$1
maven_repo=${staging_repo:-https://repo1.maven.org/maven2}

# Refresh m2 distro project
rm -rf ${rootdir}/build/m2
mkdir -p ${rootdir}/build/m2

if [ -z "${local_runtime_dir}" ]; then
  # Remote M2 distro
  if [ ! -z $staging_repo ]; then
    # Change the settings to include the staging repo if it's not already there
    echo "INFO: updating the settings staging repository"
    sed -i "s;<url>https://repository\.apache\.org/content/repositories/orgapachecamel-.*</url>;<url>$staging_repo</url>;" $location/maven-settings.xml
  fi

  #TODO: remove this check once Camel K 1.16.0 is released
  if [[ $camel_k_runtime_version != *"SNAPSHOT"* ]]; then
    echo "WARN: Package Camel K runtime artifacts temporary removed because of https://github.com/apache/camel-k-runtime/pull/928 issue"
    echo "Please, remove this check when Camel K Runtime 1.16.0 is officially released"
    exit 0
  fi
else
  # Local M2 distro
  echo "Installing local Camel K runtime $camel_k_runtime_version M2 from $local_runtime_dir (may take some minute ...)"
  mvn -q -f $local_runtime_dir/distribution clean install
fi

echo "Downloading Camel K runtime $camel_k_runtime_version M2 (may take some minute ...)"
mvn -q dependency:copy -Dartifact="org.apache.camel.k:apache-camel-k-runtime:$camel_k_runtime_version:zip:m2" \
  -Dmdep.useBaseVersion=true \
  -DoutputDirectory=${rootdir}/build/m2 \
  -s $location/maven-settings.xml \
  -Papache
# Refresh maven output dir
rm -rf ${camel_k_destination}
mkdir -p ${camel_k_destination}
unzip -q -o $PWD/build/m2/apache-camel-k-runtime-${camel_k_runtime_version}-m2.zip -d $camel_k_destination

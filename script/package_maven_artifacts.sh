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
camel_k_project_pom_location=""
maven_repo=${staging_repo:-https://repo1.maven.org/maven2}

if [ ! -z $staging_repo ]; then
  # Change the settings to include the staging repo if it's not already there
  echo "INFO: updating the settings staging repository"
  sed -i "s;<url>https://repository\.apache\.org/content/repositories/orgapachecamel-.*</url>;<url>$staging_repo</url>;" $location/maven-settings.xml
fi

if [ -z "${local_runtime_dir}" ]; then
  mvn -q dependency:copy \
    -Dartifact="org.apache.camel.k:apache-camel-k-runtime:${camel_k_runtime_version}:zip:source-release" \
    -D mdep.useBaseVersion=true \
    -Papache \
    -s $location/maven-settings.xml \
    -DoutputDirectory=$PWD/build/.
  unzip -q -o $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}-source-release.zip -d $PWD/build
  rm $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}-source-release.zip

  camel_k_project_pom_location=$PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}/pom.xml
else
  # Take the dependencies from a local development environment
  camel_k_runtime_source=${local_runtime_dir}
  camel_k_runtime_source_version=$(mvn $options -f $camel_k_runtime_source/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout)

  if [ "$camel_k_runtime_version" != "$camel_k_runtime_source_version" ]; then
      echo "Misaligned version. You're building Camel K project using $camel_k_runtime_version but trying to bundle dependencies from local Camel K runtime $camel_k_runtime_source_version"
      exit 2
  fi

  camel_k_project_pom_location=${local_runtime_dir}/pom.xml
fi

echo "Extracting Camel K runtime $camel_k_runtime_version dependencies... (may take some minutes to download)"
mvn -q \
  -f $camel_k_project_pom_location \
  dependency:copy-dependencies \
  -DincludeScope=runtime \
  -Dmdep.copyPom=true \
  -DoutputDirectory=$camel_k_destination \
  -Dmdep.useRepositoryLayout=true \
  -Papache \
  -s $location/maven-settings.xml
    
if [ -z "${local_runtime_dir}" ]; then
  rm -rf $PWD/build/apache-camel-k-runtime-${camel_k_runtime_version}
fi 
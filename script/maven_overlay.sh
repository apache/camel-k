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

if [ "$#" -lt 2 ]; then
  echo "usage: $0 [-s <staging repository>] [-d <local Camel K runtime project directory>] <Camel K runtime version> <output directory>"
  exit 1
fi

options=""
if [ "$CI" = "true" ]; then
  options="--batch-mode"
fi

runtime_version=$1
output_dir=$2

if [ ! -z $staging_repo ]; then
  # Change the settings to include the staging repo if it's not already there
  echo "INFO: updating the settings staging repository"
  sed -i "s;<url>https://repository\.apache\.org/content/repositories/orgapachecamel-.*</url>;<url>$staging_repo</url>;" $location/maven-settings.xml
fi

if [ ! -z "$local_runtime_dir" ]; then
    # Take the dependencies from a local development environment
    camel_k_runtime_source_version=$(mvn -f $local_runtime_dir/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout)

    if [ "$runtime_version" != "$camel_k_runtime_source_version" ]; then
        echo "WARNING! You're building Camel K project using $runtime_version but trying to bundle dependencies from local Camel K runtime $camel_k_runtime_source_version"
    fi

    mvn -q \
      $options \
      -am \
      -f $local_runtime_dir/support/camel-k-maven-logging/pom.xml \
      install
fi

mvn -q \
  $options \
  dependency:copy \
  -Dartifact=org.apache.camel.k:camel-k-maven-logging:${runtime_version}:zip \
  -D mdep.useBaseVersion=true \
  -DoutputDirectory=${rootdir}/${output_dir} \
  -Papache \
  -s $location/maven-settings.xml

unzip -q -o ${rootdir}/${output_dir}/camel-k-maven-logging-${runtime_version}.zip -d ${rootdir}/${output_dir}
rm -f camel-k-maven-logging-${runtime_version}.zip


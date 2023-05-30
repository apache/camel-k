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

script_dir=$(dirname $0)
rootdir=$(realpath ${script_dir}/../)

while getopts "d:s:" opt; do
  case "${opt}" in
  d)
    local_runtime_dir="${OPTARG}"
    ;;
  s)
    staging_repo="${OPTARG}"
    ;;
  *) ;;
  esac
done
shift $((OPTIND - 1))

if [ "$#" -lt 1 ]; then
  echo "usage: $0 [-s <staging repository>] [-d <local Camel K runtime project directory>] <Camel K runtime version>"
  exit 1
fi

runtime_version=$1

if [ -z "${local_runtime_dir}" ]; then
  # Remote M2 distro
  if [ ! -z $staging_repo ]; then
    # Change the settings to include the staging repo if it's not already there
    echo "INFO: updating the settings staging repository to $staging_repo"
    sed -i "s;<url>https://repository\.apache\.org/content/repositories/orgapachecamel-.*</url>;<url>$staging_repo</url>;" $script_dir/maven-settings.xml
  fi

  echo "INFO: Retrieving camel-k-catalog:$runtime_version:yaml:catalog from the apache repository"
  mvn -q dependency:copy -Dartifact="org.apache.camel.k:camel-k-catalog:$runtime_version:yaml:catalog" \
    -Dmdep.useBaseVersion=true \
    -DoutputDirectory=${rootdir}/resources/ \
    -s $script_dir/maven-settings.xml \
    -Papache

  if [ -f "${rootdir}/resources/camel-k-catalog-${runtime_version}-catalog.yaml" ]; then
    mv ${rootdir}/resources/camel-k-catalog-"${runtime_version}"-catalog.yaml ${rootdir}/resources/camel-catalog-"${runtime_version}".yaml
  fi

else

  local_camel_catalog="support/camel-k-catalog/target/camel-k-catalog-${runtime_version}.yaml"

  if [ -f "${local_runtime_dir}/$local_camel_catalog" ]; then
    echo "INFO: Copy Existing ${local_runtime_dir}/$local_camel_catalog to resources/camel-catalog-${runtime_version}.yaml"
    cp "${local_runtime_dir}/$local_camel_catalog" ${rootdir}/resources/camel-catalog-"${runtime_version}".yaml
  else
    echo "INFO: Build and copy camel-k-catalog from local ${local_runtime_dir} to resources/camel-catalog-${runtime_version}.yaml"
    mvn -o -q -f "${local_runtime_dir}/support/camel-k-catalog/pom.xml" install
    if [ -f "${local_runtime_dir}/$local_camel_catalog" ]; then
      cp "${local_runtime_dir}/$local_camel_catalog" ${rootdir}/resources/camel-catalog-"${runtime_version}".yaml
    fi
  fi

fi

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
rootdir=$location/../

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <Camel K runtime version> [<staging repository>]"
  exit 1
fi
runtime_version="$1"

if [ ! -z $2 ]; then
  # Change the settings to include the staging repo if it's not already there
  echo "INFO: updating the settings staging repository"
  sed -i "s;<url>https://repository\.apache\.org/content/repositories/orgapachecamel-.*</url>;<url>$2</url>;" $location/maven-settings.xml
fi

# Refresh catalog sets. We can clean any leftover as well.
rm -f ${rootdir}/resources/camel-catalog-*

mvn -q dependency:copy -Dartifact="org.apache.camel.k:camel-k-catalog:$runtime_version:yaml:catalog" \
  -DoutputDirectory=${rootdir}/resources/ \
  -s $location/maven-settings.xml \
  -Papache-snapshots
if [ -f "${rootdir}/resources/camel-k-catalog-${runtime_version}-catalog.yaml" ]; then
    mv ${rootdir}/resources/camel-k-catalog-"${runtime_version}"-catalog.yaml ${rootdir}/resources/camel-catalog-"${runtime_version}".yaml
elif [[ $runtime_version == *"SNAPSHOT" ]]; then
    # when using SNAPSHOT versions and a snapshot repository the downloaded artifact has a timestamp in place of the SNAPSHOT word
    # we should replace this timestamp with the SNAPSHOT word
    only_version=$(echo $runtime_version|sed 's/-SNAPSHOT//g')
    cat_file=$(ls -lrt ${rootdir}/resources/camel-k-catalog-"${only_version}"*-catalog.yaml|tail -1|sed 's/.*resources\///g')
    mv ${rootdir}/resources/$cat_file ${rootdir}/resources/camel-catalog-"${runtime_version}".yaml
fi


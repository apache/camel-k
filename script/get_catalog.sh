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
rootdir=$location/../

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <Camel K runtime version> [<staging repository>]"
  exit 1
fi

if [ -z $2 ]; then
  mvn -q dependency:copy -Dartifact="org.apache.camel.k:camel-k-catalog:$1:yaml:catalog" -DoutputDirectory=${rootdir}/resources/
  mv ${rootdir}/resources/camel-k-catalog-$1-catalog.yaml ${rootdir}/resources/camel-catalog-$1.yaml
else
  # TODO: fix this workaround to use the above mvn statement with the staging repository as well
  echo "INFO: extracting a catalog from staging repository $2"
  wget -q $2/org/apache/camel/k/camel-k-catalog/$1/camel-k-catalog-$1-catalog.yaml -O ${rootdir}/resources/camel-catalog.yaml

  if [ -s ${rootdir}/resources/camel-catalog.yaml ]; then
    # the extracted catalog file is not empty
    mv ${rootdir}/resources/camel-catalog.yaml ${rootdir}/resources/camel-catalog-$1.yaml
  else
    # the extracted catalog file is empty - some error in staging repository
    echo "WARNING: could not extract catalog from staging repository $2"
    rm ${rootdir}/resources/camel-catalog.yaml
  fi
fi



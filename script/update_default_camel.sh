#!/bin/bash

# ---------------------------------------------------------------------------
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
# ---------------------------------------------------------------------------

set -e

location=$(dirname $0)

rm -rf /tmp/camel-k-runtime
git clone --depth 1 https://github.com/apache/camel-k-runtime.git /tmp/camel-k-runtime
ck_runtime_version=$(mvn help:evaluate -Dexpression=project.version -q -DforceStdout -f /tmp/camel-k-runtime/pom.xml)
echo "INFO: last Camel K runtime version set at $ck_runtime_version"
sed -i "s/^DEFAULT_RUNTIME_VERSION := .*$/DEFAULT_RUNTIME_VERSION := $ck_runtime_version/" $location/Makefile
camel_version=$(grep -oPm1 "(?<=<camel-version>)[^<]+" /tmp/camel-k-runtime/pom.xml)

rm -rf /tmp/camel-kamelets
git clone https://github.com/apache/camel-kamelets.git /tmp/camel-kamelets
pushd /tmp/camel-kamelets
echo "INFO: Looking a suitable Kamelet tag for $camel_version camel version"
kamelets_tag=$(git tag | grep $camel_version | sort -r | head -n 1)
popd
echo "INFO: Kamelets version set at $kamelets_tag"
sed -i "s/^KAMELET_CATALOG_REPO_TAG := .*$/KAMELET_CATALOG_REPO_TAG := $kamelets_tag/" $location/Makefile

rm -rf /tmp/camel-k-runtime/ /tmp/camel-kamelets
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

set -e

location=$(dirname $0)
rootdir=$location/../
outputLocation=${rootdir}build/_offline

if [ "$#" -lt 3 ]; then
  echo "usage: $0 <Camel K runtime version> --with <path/to/maven/version>"
  exit 1
fi

# Non reproducible builds: we must use the exact maven version used by the operator
# Change the mvnCmd variable to include the maven version you're willing to use ie, /usr/share/apache-maven-3.8.6/bin/mvn
if [ ! "$2" == "--with" ]; then
  echo "usage: $0 <Camel K runtime version> --with <path/to/maven/version>"
  exit 2
fi

runtime_version="$1"
mvnCmd="$3"

echo "WARN: Running the script with the following maven version. Make sure maven version matches the target!"
$mvnCmd --version | grep "Apache Maven"

echo "INFO: downloading catalog for Camel K runtime $1..."
${location}/get_catalog.sh $1 $2

catalog="$rootdir/resources/camel-catalog-$runtime_version.yaml"
ckr_version=$(yq .spec.runtime.version $catalog)
cq_version=$(yq '.spec.runtime.metadata."camel-quarkus.version"' $catalog)
quarkus_version=$(yq '.spec.runtime.metadata."quarkus.version"' $catalog)

echo "INFO: configuring offline dependencies for Camel K Runtime $ckr_version, Camel Quarkus $cq_version and Quarkus version $quarkus_version"

echo "INFO: preparing a base project to download maven dependencies..."

$mvnCmd -q clean package \
    -f $location/camel-k-runtime-archetype/pom.xml \
    -Dmaven.repo.local=$outputLocation \
    -DRUNTIME_VERSION_CMD=$ckr_version \
    -DQUARKUS_VERSION_CMD=$quarkus_version \
    -s $location/maven-settings.xml

sed 's/- //g' $catalog | grep "groupId\|artifactId" | paste -d " "  - - | awk '{print $2,":",$4}' | tr -d " " | sort | uniq > /tmp/ck.dependencies

dependencies=$(cat /tmp/ck.dependencies)

# TODO: include this dependency in the catalog
$mvnCmd -q dependency:get -Dartifact=org.apache.camel.k:camel-k-runtime-bom:$runtime_version:pom -Dmaven.repo.local=$outputLocation -s $location/maven-settings.xml

for d in $dependencies
do
    mvn_dep=""
    mvn_dep_deployment=""
    if [[ $d == org.apache.camel.quarkus* ]]; then
        mvn_dep="$d:$cq_version"
        mvn_dep_deployment="$d-deployment:$cq_version"
    elif [[ $d == org.apache.camel.k* ]]; then
        mvn_dep="$d:$ckr_version"
    else
        echo "WARN: cannot parse $d kind of dependency (likely it misses the version), skipping as it should be imported transitively. If not, add manually to your bundle."
        continue
    fi
    echo "INFO: downloading $mvn_dep and its transitive dependencies..."
    $mvnCmd -q dependency:get -Dartifact=$mvn_dep -Dmaven.repo.local=$outputLocation -s $location/maven-settings.xml
    if [[ ! $mvn_dep_deployment == "" ]]; then
        $mvnCmd -q dependency:get -Dartifact=$mvn_dep_deployment -Dmaven.repo.local=$outputLocation -s $location/maven-settings.xml
    fi
done

# we can bundle into a single archive now
echo "INFO: building ${rootdir}build/camel-k-runtime-$runtime_version-maven-offline.tar.gz archive..."
pushd $outputLocation
tar -czf ../camel-k-runtime-$runtime_version-maven-offline.tar.gz *
popd
echo "INFO: deleting cached dependencies..."
rm -rf $outputLocation
echo "Success: your bundled set of offline dependencies is available in camel-k-runtime-$runtime_version-maven-offline.tar.gz file."

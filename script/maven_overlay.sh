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

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <Camel K runtime version> <output directory> [<staging repository>] [<local Camel K runtime project directory>]"
  exit 1
fi

options=""
if [ "$CI" = "true" ]; then
  options="--batch-mode"
fi

runtime_version=$1
output_dir=$2
staging_repo=${3:-}
local_runtime_dir=${4:-}

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
    else
        local_runtime_dir="${staging_repo}"
        staging_repo=""
    fi
fi


if [ -z "$local_runtime_dir" ]; then
    mvn -q \
      $options \
      dependency:copy \
      -Dartifact=org.apache.camel.k:camel-k-maven-logging:${runtime_version}:zip \
      -DoutputDirectory=${rootdir}/${output_dir}

    mv ${rootdir}/${output_dir}/camel-k-maven-logging-${runtime_version}.zip ${rootdir}/${output_dir}/camel-k-maven-logging.zip
else
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

    mvn -q \
      $options \
      dependency:copy \
      -Dartifact=org.apache.camel.k:camel-k-maven-logging:$camel_k_runtime_source_version:zip \
      -DoutputDirectory=${rootdir}/${output_dir}

    mv ${rootdir}/${output_dir}/camel-k-maven-logging-$camel_k_runtime_source_version.zip ${rootdir}/${output_dir}/camel-k-maven-logging.zip
fi

unzip -q -o ${rootdir}/${output_dir}/camel-k-maven-logging.zip -d ${rootdir}/${output_dir}

if [ -n "$staging_repo" ]; then
    rm ${rootdir}/settings.xml
fi

rm ${rootdir}/${output_dir}/camel-k-maven-logging.zip

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

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <output directory>"
  exit 1
fi

options=""
if [ "$CI" = "true" ]; then
  options="--batch-mode"
fi

output_dir=$1

mvn $options -f java/maven-logging/pom.xml clean package
cp java/maven-logging/target/maven-overlay/*.jar ${output_dir}
cp java/maven-logging/src/main/resources/logback.xml ${output_dir}

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
cd $location/..

version=$1
repo=$2

[ -d "./tmpschema" ] && rm -r ./tmpschema
mkdir  tmpschema

./mvnw dependency:copy \
  -f build/maven/pom-catalog.xml \
  -Dartifact=org.apache.camel.k:camel-k-loader-yaml-impl:$version:json:json-schema \
  -DoutputDirectory=../../tmpschema \
  -Dmdep.stripVersion \
  -Druntime.version=$1 \
  -Dstaging.repo=$repo

schema=./tmpschema/camel-k-loader-yaml-impl-json-schema.json

go run ./cmd/util/json-schema-gen ./deploy/crd-kamelet.yaml $schema .spec.flow false ./docs/modules/ROOT/assets/attachments/schema/kamelet-schema.json
go run ./cmd/util/json-schema-gen ./deploy/crd-integration.yaml $schema .spec.flows true ./docs/modules/ROOT/assets/attachments/schema/integration-schema.json

rm -r ./tmpschema

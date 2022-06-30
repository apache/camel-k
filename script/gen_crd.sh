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

location=$(dirname "$0")
apidir=$location/../pkg/apis/camel

cd "$apidir"
$CONTROLLER_GEN crd \
  paths=./... \
  output:crd:artifacts:config=../../../config/crd/bases \
  output:crd:dir=../../../config/crd/bases \
  crd:crdVersions=v1

# cleanup working directory in $apidir
rm -rf ./config

# to root
cd ../../../

deploy_crd_file() {
  source=$1

  # Make a copy to serve as the base for post-processing
  cp "$source" "${source}.orig"

  # Post-process source
  cat ./script/headers/yaml.txt > "$source"
  echo "" >> "$source"
  sed -n '/^---/,/^status/p;/^status/q' "${source}.orig" \
    | sed '1d;$d' \
    | sed '/creationTimestamp:/a\  labels:\n    app: camel-k' >> "$source"

  for dest in "${@:2}"; do
    cp "$source" "$dest"
  done

  # Remove the copy as no longer required
  rm -f "${source}.orig"
}

deploy_crd() {
  name=$1
  plural=$2

  deploy_crd_file ./config/crd/bases/camel.apache.org_"$plural".yaml \
    ./helm/camel-k/crds/crd-"$name".yaml
}

deploy_crd build builds
deploy_crd camel-catalog camelcatalogs
deploy_crd integration integrations
deploy_crd integration-kit integrationkits
deploy_crd integration-platform integrationplatforms
deploy_crd kamelet kamelets
deploy_crd kamelet-binding kameletbindings

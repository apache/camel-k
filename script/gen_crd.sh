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
apidir=$location/../pkg/apis/camel

echo "Generating CRDs..."

cd $apidir
$CONTROLLER_GEN crd paths=./... output:crd:artifacts:config=false output:crd:dir=../../../deploy/crds crd:crdVersions=v1

# cleanup
rm -r ./config

# to root
cd ../../../

version=$(make -s get-version | tr '[:upper:]' '[:lower:]')
echo "Version for OLM: $version"

deploy_crd_file() {
  source=$1

  for dest in ${@:2}; do
    cat ./script/headers/yaml.txt > $dest
    echo "" >> $dest
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
      cat $source | sed -n '/^---/,/^status/p;/^status/q' \
        | sed '1d;$d' \
        | sed 's/^metadata:/metadata:\n  labels:\n    app: "camel-k"/' >> $dest
    elif [[ "$OSTYPE" == "darwin"* ]]; then
      # Mac OSX
      cat $source | sed -n '/^---/,/^status/p;/^status/q' \
        | sed '1d;$d' \
        | sed $'s/^metadata:/metadata:\\\n  labels:\\\n    app: "camel-k"/' >> $dest
    fi
  done

}

deploy_crd() {
  name=$1
  plural=$2

  deploy_crd_file ./deploy/crds/camel.apache.org_$plural.yaml \
    ./deploy/crd-$name.yaml \
    ./helm/camel-k/crds/crd-$name.yaml \
    ./deploy/olm-catalog/camel-k-dev/$version/$plural.camel.apache.org.crd.yaml \
    ./config/crd/bases/camel.apache.org_$plural.yaml
}

deploy_crd build builds
deploy_crd camel-catalog camelcatalogs
deploy_crd integration integrations
deploy_crd integration-kit integrationkits
deploy_crd integration-platform integrationplatforms
deploy_crd kamelet kamelets
deploy_crd kamelet-binding kameletbindings

rm -r ./deploy/crds

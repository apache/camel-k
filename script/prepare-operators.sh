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

if [ "$#" -lt 1 ]; then
    echo "usage: $0 prepare-operators release-version"
    exit 1
fi

location=$(dirname $0)
version=$1

cd bundle/

mkdir -p k8s-operatorhub/$1/manifests/
mkdir -p k8s-operatorhub/$1/metadata/
mkdir -p k8s-operatorhub/$1/tests/scorecard/
mkdir -p openshift-ecosystem/$1/manifests/
mkdir -p openshift-ecosystem/$1/metadata/
mkdir -p openshift-ecosystem/$1/tests/scorecard/

cp ./manifests/camel.apache.org_builds.yaml k8s-operatorhub/$1/manifests/builds.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_camelcatalogs.yaml k8s-operatorhub/$1/manifests/camelcatalogs.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrationkits.yaml k8s-operatorhub/$1/manifests/integrationkits.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrationplatforms.yaml k8s-operatorhub/$1/manifests/integrationplatforms.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrations.yaml k8s-operatorhub/$1/manifests/integrations.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_kameletbindings.yaml k8s-operatorhub/$1/manifests/kameletbindings.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_kamelets.yaml k8s-operatorhub/$1/manifests/kamelets.camel.apache.org.crd.yaml
cp ./manifests/camel-k.clusterserviceversion.yaml k8s-operatorhub/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
cp ./metadata/annotations.yaml k8s-operatorhub/$1/metadata/annotations.yaml
cp ./tests/scorecard/config.yaml k8s-operatorhub/$1/tests/scorecard/config.yaml

cp ./manifests/camel.apache.org_builds.yaml openshift-ecosystem/$1/manifests/builds.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_camelcatalogs.yaml openshift-ecosystem/$1/manifests/camelcatalogs.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrationkits.yaml openshift-ecosystem/$1/manifests/integrationkits.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrationplatforms.yaml openshift-ecosystem/$1/manifests/integrationplatforms.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_integrations.yaml openshift-ecosystem/$1/manifests/integrations.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_kameletbindings.yaml openshift-ecosystem/$1/manifests/kameletbindings.camel.apache.org.crd.yaml
cp ./manifests/camel.apache.org_kamelets.yaml openshift-ecosystem/$1/manifests/kamelets.camel.apache.org.crd.yaml
cp ./manifests/camel-k.clusterserviceversion.yaml openshift-ecosystem/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
cp ./metadata/annotations.yaml openshift-ecosystem/$1/metadata/annotations.yaml
cp ./tests/scorecard/config.yaml openshift-ecosystem/$1/tests/scorecard/config.yaml

# Starting sed to replace operator

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  sed -i 's/camel-k.v/camel-k-operator.v/g' k8s-operatorhub/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
  sed -i 's/camel-k.v/camel-k-operator.v/g' openshift-ecosystem/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
elif [[ "$OSTYPE" == "darwin"* ]]; then
  # Mac OSX
  sed -i '' 's/camel-k.v/camel-k-operator.v/g' k8s-operatorhub/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
  sed -i '' 's/camel-k.v/camel-k-operator.v/g' openshift-ecosystem/$1/manifests/camel-k.v$1.clusterserviceversion.yaml
fi

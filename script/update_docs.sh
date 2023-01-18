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

echo "Scraping information from Makefile"
RUNTIME_VERSION=$(grep '^RUNTIME_VERSION := ' Makefile | sed 's/^.* \?= //')

CATALOG="$location/../resources/camel-catalog-$RUNTIME_VERSION.yaml"
# This script requires the catalog to be available (via make build-resources for instance)
if [ ! -f $CATALOG ]; then
    echo "❗ catalog not available. Make sure to download it before calling this script."
    exit 1
fi

KAMELET_CATALOG_REPO_TAG=$(grep '^KAMELET_CATALOG_REPO_TAG := ' Makefile | sed 's/^.* \?= //')
KAMELETS_VERSION=$(echo $KAMELET_CATALOG_REPO_TAG | sed 's/^.//')
if [[ "$KAMELET_CATALOG_REPO_TAG" == "main" ]]; then
    KAMELETS_VERSION="latest"
    KAMELETS_DOCS_VERSION="next"
else
    re="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)$"
    if ! [[ $KAMELETS_VERSION =~ $re ]]; then
        echo "❗ argument must match semantic version: $KAMELETS_VERSION"
        exit 1
    fi
    KAMELETS_DOCS_VERSION="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.x"
fi
BUILDAH_VERSION=$(grep '^BUILDAH_VERSION := ' Makefile | sed 's/^.* \?= //')
KANIKO_VERSION=$(grep '^KANIKO_VERSION := ' Makefile | sed 's/^.* \?= //')
KUSTOMIZE_VERSION=$(grep '^KUSTOMIZE_VERSION := ' Makefile | sed 's/^.* \?= //' | sed 's/^.//')

echo "Camel K Runtime version: $RUNTIME_VERSION"
echo "Kamelets version: $KAMELETS_VERSION"
echo "Buildah version: $BUILDAH_VERSION"
echo "Kaniko version: $KANIKO_VERSION"
echo "Kustomize version: $KUSTOMIZE_VERSION"

yq -i ".asciidoc.attributes.buildah-version = \"$BUILDAH_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.kaniko-version = \"$KANIKO_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.kustomize-version = \"$KUSTOMIZE_VERSION\"" $location/../docs/antora.yml

echo "Scraping information from catalog available at: $CATALOG"
RUNTIME_VERSION=$(yq '.spec.runtime.version' $CATALOG)
CAMEL_VERSION=$(yq '.spec.runtime.metadata."camel.version"' $CATALOG)
re="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)$"
if ! [[ $CAMEL_VERSION =~ $re ]]; then
    echo "❗ argument must match semantic version: $CAMEL_VERSION"
    exit 1
fi
CAMEL_DOCS_VERSION="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.x"
CAMEL_QUARKUS_VERSION=$(yq '.spec.runtime.metadata."camel-quarkus.version"' $CATALOG)
re="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)$"
if ! [[ $CAMEL_QUARKUS_VERSION =~ $re ]]; then
    echo "❗ argument must match semantic version: $CAMEL_QUARKUS_VERSION"
    exit 1
fi
CAMEL_QUARKUS_DOCS_VERSION="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.x"
QUARKUS_VERSION=$(yq '.spec.runtime.metadata."quarkus.version"' $CATALOG)

echo "Camel K Runtime version: $RUNTIME_VERSION"
echo "Camel version: $CAMEL_VERSION"
echo "Camel Quarkus version: $CAMEL_QUARKUS_VERSION"
echo "Quarkus version: $QUARKUS_VERSION"

yq -i ".asciidoc.attributes.camel-k-runtime-version = \"$RUNTIME_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-version = \"$CAMEL_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-docs-version = \"$CAMEL_DOCS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-quarkus-version = \"$CAMEL_QUARKUS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-quarkus-docs-version = \"$CAMEL_QUARKUS_DOCS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.quarkus-version = \"$QUARKUS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-kamelets-version = \"$KAMELETS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-kamelets-docs-version = \"$KAMELETS_DOCS_VERSION\"" $location/../docs/antora.yml

echo "Scraping information from go.mod"
KNATIVE_API_VERSION=$(grep '^.*knative.dev/eventing ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
KUBE_API_VERSION=$(grep '^.*k8s.io/api ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
OPERATOR_FWK_API_VERSION=$(grep '^.*github.com/operator-framework/api ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
SERVICE_BINDING_OP_VERSION=$(grep '^.*github.com/redhat-developer/service-binding-operator ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
PROMETHEUS_OP_VERSION=$(grep '^.*github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')

echo "Kubernetes API version: $KUBE_API_VERSION"
echo "Operator Framework API version: $OPERATOR_FWK_API_VERSION"
echo "Knative API version: $KNATIVE_API_VERSION"
echo "Service Binding Operator version: $SERVICE_BINDING_OP_VERSION"
echo "Prometheus Operator version: $PROMETHEUS_OP_VERSION"

yq -i ".asciidoc.attributes.kubernetes-api-version = \"$KUBE_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.operator-fwk-api-version = \"$OPERATOR_FWK_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.knative-api-version = \"$KNATIVE_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.service-binding-op-version = \"$SERVICE_BINDING_OP_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.prometheus-op-version = \"$PROMETHEUS_OP_VERSION\"" $location/../docs/antora.yml

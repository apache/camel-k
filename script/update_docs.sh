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
RUNTIME_VERSION=$(grep '^RUNTIME_VERSION := ' Makefile | sed 's/^.* \?= //')
KAMELETS_VERSION=$(grep '^KAMELET_CATALOG_REPO_BRANCH := ' Makefile | sed 's/^.* \?= //' | sed 's/^.//')
re="^([[:digit:]]+)\.([[:digit:]]+)\.([[:digit:]]+)$"
if ! [[ $KAMELETS_VERSION =~ $re ]]; then
    echo "❗ argument must match semantic version: $KAMELETS_VERSION"
    exit 1
fi
KAMELETS_DOCS_VERSION="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.x"

CATALOG="$location/../resources/camel-catalog-$RUNTIME_VERSION.yaml"
# This script requires the catalog to be available (via make build-resources for instance)
if [ ! -f $CATALOG ]; then
    echo "❗ catalog not available. Make sure to download it before calling this script."
    exit 1
fi
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
echo "Kamelets version: $KAMELETS_VERSION"

yq -i ".asciidoc.attributes.camel-k-runtime-version = \"$RUNTIME_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-version = \"$CAMEL_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-docs-version = \"$CAMEL_DOCS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-quarkus-version = \"$CAMEL_QUARKUS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-quarkus-docs-version = \"$CAMEL_QUARKUS_DOCS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.quarkus-version = \"$QUARKUS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-kamelets-version = \"$KAMELETS_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.camel-kamelets-docs-version = \"$KAMELETS_DOCS_VERSION\"" $location/../docs/antora.yml
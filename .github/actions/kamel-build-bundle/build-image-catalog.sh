#!/bin/bash

# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

####
#
# Builds the kamel bundle index image
#
####

set -e

while getopts ":c:i:x:" opt; do
  case "${opt}" in
    c)
      CATALOG_SOURCE_NAMESPACE=${OPTARG}
      ;;
    i)
      IMAGE_NAMESPACE=${OPTARG}
      ;;
    x)
      BUNDLE_IMAGE_INDEX=${OPTARG}
      ;;
    :)
      echo "ERROR: Option -$OPTARG requires an argument"
      exit 1
      ;;
    \?)
      echo "ERROR: Invalid option -$OPTARG"
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

if [ -z "${CATALOG_SOURCE_NAMESPACE}" ]; then
  echo "No catalog source namespace defined ... skipping catalog source creation"
  exit 0
fi

if [ -z "${IMAGE_NAMESPACE}" ]; then
  echo "Error: image-namespace not defined"
  exit 1
fi

if [ -z "${BUNDLE_IMAGE_INDEX}" ]; then
  echo "Error: build-bundle-image-bundle-index not defined"
  exit 1
fi

kubectl get ns ${CATALOG_SOURCE_NAMESPACE} &> /dev/null
if [ $? != 0 ]; then
  echo "Error: Catalog source cannot be created as namespace ${CATALOG_SOURCE_NAMESPACE} does not exist."
  exit 1
fi

export BUILD_CATALOG_SOURCE="camel-k-test-source"
echo "Setting build-bundle-catalog-source-name to ${BUILD_CATALOG_SOURCE}"
echo "::set-output name=build-bundle-catalog-source-name::${BUILD_CATALOG_SOURCE}"

cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${BUILD_CATALOG_SOURCE}
  namespace: ${IMAGE_NAMESPACE}
spec:
  displayName: OLM upgrade test Catalog
  image: ${BUNDLE_IMAGE_INDEX}
  sourceType: grpc
  publisher: grpc
  updateStrategy:
    registryPoll:
      interval: 1m0s
EOF

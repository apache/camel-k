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

while getopts ":b:c:i:x:" opt; do
  case "${opt}" in
    b)
      CATALOG_SOURCE_NAME=${OPTARG}
      ;;
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

if [ -z "${CATALOG_SOURCE_NAME}" ]; then
  echo "No catalog source name defined ... skipping catalog source creation"
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

cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${CATALOG_SOURCE_NAME}
  namespace: ${CATALOG_SOURCE_NAMESPACE}
spec:
  displayName: OLM upgrade test Catalog
  image: ${BUNDLE_IMAGE_INDEX}
  sourceType: grpc
  publisher: grpc
  updateStrategy:
    registryPoll:
      interval: 1m0s
EOF

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
# Execute the upgrade tests
#
####

set -e

while getopts ":b:d:l:n:s:v:x:" opt; do
  case "${opt}" in
    b)
      KAMEL_BINARY=${OPTARG}
      ;;
    d)
      BUNDLE_INDEX_IMAGE=${OPTARG}
      ;;
    l)
      REGISTRY_PULL_HOST=${OPTARG}
      ;;
    n)
      IMAGE_NAME=${OPTARG}
      ;;
    s)
      REGISTRY_INSECURE=${OPTARG}
      ;;
    v)
      IMAGE_VERSION=${OPTARG}
      ;;
    x)
      SAVE_FAILED_TEST_NS=${OPTARG}
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

if [ -z "${IMAGE_NAME}" ]; then
  echo "Error: local-image-name not defined"
  exit 1
fi

if [ -z "${IMAGE_VERSION}" ]; then
  echo "Error: local-image-version not defined"
  exit 1
fi

if [ -z "${KAMEL_BINARY}" ]; then
  echo "Error: kamel-binary not defined"
  exit 1
fi

if [ -z "${BUNDLE_INDEX_IMAGE}" ]; then
  echo "Error: bundle-index-image not defined"
  exit 1
fi

if [ -z "${REGISTRY_PULL_HOST}" ]; then
  echo "Error: image-registry-pull-host not defined"
  exit 1
fi

if [ -z "${REGISTRY_INSECURE}" ]; then
  echo "Error: image-registry-insecure not defined"
  exit 1
fi

# Use the last released Kamel CLI
export RELEASED_KAMEL_BIN=${KAMEL_BINARY}

echo "Kamel version: $(${RELEASED_KAMEL_BIN} version)"

# Cluster environment
export CUSTOM_IMAGE=${IMAGE_NAME}
export CUSTOM_VERSION=${IMAGE_VERSION}

# Configure install options
export KAMEL_INSTALL_MAVEN_REPOSITORIES=$(make get-staging-repo)
export KAMEL_INSTALL_REGISTRY=${REGISTRY_PULL_HOST}
export KAMEL_INSTALL_REGISTRY_INSECURE=${REGISTRY_INSECURE}

# Will only have an effect if olm=false
# since, for OLM, the csv determines the policy
# (see kamel-build-bundle/build-bundle-image.sh)
export KAMEL_INSTALL_OPERATOR_IMAGE_PULL_POLICY="Always"

# Despite building a bundle we don't want it installed immediately so no OLM_INDEX_BUNDLE var

# Configure test options
export CAMEL_K_PREV_IIB=quay.io/operatorhubio/catalog:latest
export CAMEL_K_NEW_IIB=${BUNDLE_INDEX_IMAGE}
export CAMEL_K_PREV_UPGRADE_CHANNEL=${PREV_XY_CHANNEL}
export CAMEL_K_NEW_UPGRADE_CHANNEL=${NEW_XY_CHANNEL}
export KAMEL_K_TEST_RELEASE_VERSION=$(make get-last-released-version)
export KAMEL_K_TEST_OPERATOR_CURRENT_IMAGE=${CUSTOM_IMAGE}:${CUSTOM_VERSION}
export CAMEL_K_TEST_SAVE_FAILED_TEST_NAMESPACE=${SAVE_FAILED_TEST_NS}

# Then run integration tests
DO_TEST_PREBUILD=false make test-upgrade

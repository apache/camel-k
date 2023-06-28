#!/bin/bash

# ---------------------------------------------------------------------------
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
# ---------------------------------------------------------------------------

####
#
# Execute the knative-yaks tests
#
####

set -e

while getopts ":b:c:i:l:n:q:s:v:x:" opt; do
  case "${opt}" in
    b)
      BUILD_CATALOG_SOURCE_NAME=${OPTARG}
      ;;
    c)
      BUILD_CATALOG_SOURCE_NAMESPACE=${OPTARG}
      ;;
    i)
      IMAGE_NAMESPACE=${OPTARG}
      ;;
    l)
      REGISTRY_PULL_HOST=${OPTARG}
      ;;
    n)
      IMAGE_NAME=${OPTARG}
      ;;
    q)
      LOG_LEVEL=${OPTARG}
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

if [ -z "${IMAGE_NAMESPACE}" ]; then
  echo "Error: image-namespace not defined"
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

# Cluster environment
export CUSTOM_IMAGE=${IMAGE_NAME}
export CUSTOM_VERSION=${IMAGE_VERSION}

#
# If bundle has been built and installed then use it
#
if [ -n "${BUILD_CATALOG_SOURCE_NAMESPACE}" ]; then
  export KAMEL_INSTALL_OLM_SOURCE=${BUILD_CATALOG_SOURCE_NAME}
  export KAMEL_INSTALL_OLM_SOURCE_NAMESPACE=${BUILD_CATALOG_SOURCE_NAMESPACE}
  export KAMEL_INSTALL_OLM_CHANNEL="${NEW_XY_CHANNEL}"
fi

export KAMEL_INSTALL_MAVEN_REPOSITORIES=$(make get-staging-repo)
export KAMEL_INSTALL_REGISTRY=${REGISTRY_PULL_HOST}
export KAMEL_INSTALL_REGISTRY_INSECURE=${REGISTRY_INSECURE}
export KAMEL_INSTALL_OPERATOR_IMAGE=${CUSTOM_IMAGE}:${CUSTOM_VERSION}

# Will only have an effect if olm=false
# since, for OLM, the csv determines the policy
# (see kamel-build-bundle/build-bundle-image.sh)
export KAMEL_INSTALL_OPERATOR_IMAGE_PULL_POLICY="Always"

export CAMEL_K_TEST_LOG_LEVEL="${LOG_LEVEL}"
if [ "${LOG_LEVEL}" == "debug" ]; then
  export CAMEL_K_TEST_MAVEN_CLI_OPTIONS="-X ${CAMEL_K_TEST_MAVEN_CLI_OPTIONS}"
fi
export CAMEL_K_TEST_IMAGE_NAME=${CUSTOM_IMAGE}
export CAMEL_K_TEST_IMAGE_VERSION=${CUSTOM_VERSION}
export CAMEL_K_TEST_SAVE_FAILED_TEST_NAMESPACE=${SAVE_FAILED_TEST_NS}

# Then run integration tests
yaks test e2e/yaks/common

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
# Install Camel K using the admin context
#
####

set -e

while getopts ":a:c:i:l:n:s:v:" opt; do
  case "${opt}" in
    a)
      KUBE_ADMIN_CTX=${OPTARG}
      ;;
    c)
      BUILD_CATALOG_SOURCE=${OPTARG}
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
    s)
      REGISTRY_INSECURE=${OPTARG}
      ;;
    v)
      IMAGE_VERSION=${OPTARG}
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

if [ -z "${KUBE_ADMIN_CTX}" ]; then
  echo "Error: kube-admin-user-ctx not defined"
  exit 1
fi

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
if [ -n "${BUILD_CATALOG_SOURCE}" ]; then
  export KAMEL_INSTALL_OLM_SOURCE_NAMESPACE=${IMAGE_NAMESPACE}
  export KAMEL_INSTALL_OLM_SOURCE=${BUILD_CATALOG_SOURCE}
fi

export KAMEL_INSTALL_MAVEN_REPOSITORIES=$(make get-staging-repo)
export KAMEL_INSTALL_REGISTRY=${REGISTRY_PULL_HOST}
export KAMEL_INSTALL_REGISTRY_INSECURE=${REGISTRY_INSECURE}
export KAMEL_INSTALL_OPERATOR_IMAGE=${CUSTOM_IMAGE}:${CUSTOM_VERSION}

# Will only have an effect if olm=false
# since, for OLM, the csv determines the policy.
# (see kamel-build-bundle/build-bundle-image.sh)
export KAMEL_INSTALL_OPERATOR_IMAGE_PULL_POLICY="Always"

#
# Get current context
#
ctx=$(kubectl config current-context)

#
# Need to be admin so switch to the admin context
#
kubectl config use-context "${KUBE_ADMIN_CTX}"

#
# Ensure built binary CRDs are always installed by turning off olm
#
kamel install --global --olm=false

#
# Change back to original context
#
kubectl config use-context "${ctx}"

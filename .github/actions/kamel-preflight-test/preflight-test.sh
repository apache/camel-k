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
# Execute a preflight test for checking the operator is at the correct version
#
####

set -e

while getopts ":c:i:l:n:s:v:" opt; do
  case "${opt}" in
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

#
# Create the preflight test namespace
#
NAMESPACE="preflight"
set +e
if kubectl get ns ${NAMESPACE} &> /dev/null
then
  kubectl delete ns ${NAMESPACE}
fi
set -e

kubectl create namespace ${NAMESPACE}
if [ $? != 0 ]; then
  echo "Error: failed to create the ${NAMESPACE} namespace"
  exit 1
fi

#trap "kubectl delete ns ${NAMESPACE} &> /dev/null" EXIT

# Cluster environment
export CUSTOM_IMAGE=${IMAGE_NAME}
export CUSTOM_VERSION=${IMAGE_VERSION}

#
# If bundle has been built and installed then use it
#
has_olm="false"
if [ -n "${BUILD_CATALOG_SOURCE}" ]; then
  export KAMEL_INSTALL_OLM_SOURCE_NAMESPACE=${IMAGE_NAMESPACE}
  export KAMEL_INSTALL_OLM_SOURCE=${BUILD_CATALOG_SOURCE}
  has_olm="true"
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
# Install the operator
#
kamel install -n ${NAMESPACE} --olm=${has_olm}
if [ $? != 0 ]; then
  echo "Error: kamel install returned an error."
  exit 1
fi

sleep 3

#
# Wait for the operator to be running
#
timeout=180
i=1
command="kubectl get pods -n ${NAMESPACE} 2> /dev/null | grep camel-k | grep Running &> /dev/null"

until eval "${command}"
do
  ((i++))
  if [ "${i}" -gt "${timeout}" ]; then
    echo "kamel operator not successfully installed, aborting due to ${timeout}s timeout"
    exit 1
  fi

  sleep 1
done

echo "Camel K operator up and running"

camel_operator=$(kubectl get pods -n ${NAMESPACE} | grep camel-k | awk '{print $1}')
camel_op_version=$(kubectl logs ${camel_operator} -n ${NAMESPACE} | sed -n 's/.*"Camel K Operator Version: \(.*\)"}/\1/p')
camel_op_commit=$(kubectl logs ${camel_operator} -n ${NAMESPACE} | sed -n 's/.*"Camel K Git Commit: \(.*\)"}/\1/p')

src_commit=$(git rev-parse HEAD)

#
# Test whether the versions are the same
#
if [ "${camel_op_version}" != "${IMAGE_VERSION}" ]; then
  echo "Preflight Test: Failure - Installed operator version (${camel_op_version} does not match expected version (${IMAGE_VERSION})"
  exit 1
fi

#
# Test whether the commit ids are the same
#
if [ "${camel_op_commit}" != "${src_commit}" ]; then
  echo "Preflight Test: Failure - Installed operator commit id (${camel_op_commit}) does not match expected commit id (${src_commit})"
  exit 1
fi

echo "Preflight Test: Success"

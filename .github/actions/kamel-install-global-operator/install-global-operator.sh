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
# Execute an install of a global operator
#
####

set -e

while getopts ":b:c:g:i:l:n:s:v:" opt; do
  case "${opt}" in
    b)
      BUILD_CATALOG_SOURCE_NAME=${OPTARG}
      ;;
    c)
      BUILD_CATALOG_SOURCE_NAMESPACE=${OPTARG}
      ;;
    g)
      GLOBAL_OPERATOR_NAMESPACE=${OPTARG}
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

if [ -z "${GLOBAL_OPERATOR_NAMESPACE}" ]; then
  echo "Error: global-operator-namespace not defined"
  exit 1
fi

#
# Check the global namespace
#
set +e
kubectl get ns ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null
if [ $? != 0 ]; then
  echo "Error: The ${GLOBAL_OPERATOR_NAMESPACE} namespace does not exist"
  exit 1
fi

# Cluster environment
export CUSTOM_IMAGE=${IMAGE_NAME}
export CUSTOM_VERSION=${IMAGE_VERSION}

#
# If bundle has been built and installed then use it
#
has_olm="false"
if [ -n "${BUILD_CATALOG_SOURCE_NAMESPACE}" ]; then
  #
  # Check catalog source is actually available
  #
  timeout=5
  catalog_ready=0
  until [ ${catalog_ready} -eq 1 ] || [ ${timeout} -eq 0 ]
  do
    echo "Info: Awaiting catalog source to become ready"
    let timeout=${timeout}-1

    STATE=$(kubectl get catalogsource ${BUILD_CATALOG_SOURCE_NAME} \
      -n ${BUILD_CATALOG_SOURCE_NAMESPACE} -o=jsonpath='{.status.connectionState.lastObservedState}')
    if [ "${STATE}" == "READY" ]; then
      let catalog_ready=1
      echo "Info: Catalog source is ready"
      continue
    else
      echo "Warning: catalog source status is not ready."
      if [ ${timeout} -eq 0 ]; then
        echo "Error: timeout while awaiting catalog source to start"
        exit 1
      fi
    fi

    sleep 1m
  done

  export KAMEL_INSTALL_OLM_SOURCE=${BUILD_CATALOG_SOURCE_NAME}
  export KAMEL_INSTALL_OLM_SOURCE_NAMESPACE=${BUILD_CATALOG_SOURCE_NAMESPACE}
  export KAMEL_INSTALL_OLM_CHANNEL="${NEW_XY_CHANNEL}"
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
kamel install -n ${GLOBAL_OPERATOR_NAMESPACE} --olm=${has_olm} --force --global
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
command="kubectl get pods -n ${GLOBAL_OPERATOR_NAMESPACE} 2> /dev/null | grep camel-k-operator | grep Running &> /dev/null"

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

sleep 3

i=0
while [ ${i} -lt 12 ]
do
  camel_operator=$(kubectl get pods -n ${GLOBAL_OPERATOR_NAMESPACE} | grep camel-k-operator | awk '{print $1}')
  echo "Camel K operator: ${camel_operator}"

  camel_op_version=$(kubectl logs ${camel_operator} -n ${GLOBAL_OPERATOR_NAMESPACE} | sed -n 's/.*"Camel K Operator Version: \(.*\)"}/\1/p')
  echo "Camel K operator version: ${camel_op_version}"

  camel_op_commit=$(kubectl logs ${camel_operator} -n ${GLOBAL_OPERATOR_NAMESPACE} | sed -n 's/.*"Camel K Git Commit: \(.*\)"}/\1/p')
  echo "Camel K operator commit: ${camel_op_commit}"

  if [ -n "${camel_op_version}" ] && [ -n "${camel_op_commit}" ]; then
    break
  fi

  echo "Cannot find camel-k operator version & commit information yet"
  echo "Status of pods in ${GLOBAL_OPERATOR_NAMESPACE}:"
  kubectl get pods -n ${GLOBAL_OPERATOR_NAMESPACE}

  sleep 10
  i=$[${i}+1]
done

if [ -z "${camel_op_version}" ]; then
  echo "Error: Failed to get camel-k operator version"
  exit 1
fi

if [ -z "${camel_op_commit}" ]; then
  echo "Error: Failed to get camel-k operator commit"
  exit 1
fi

src_commit=$(git rev-parse HEAD)

#
# Test whether the versions are the same
#
if [ "${camel_op_version}" != "${IMAGE_VERSION}" ]; then
  echo "Global Install Operator: Failure - Installed operator version (${camel_op_version}) does not match expected version (${IMAGE_VERSION})"
  exit 1
fi

#
# Test whether the commit ids are the same
#
if [ "${camel_op_commit}" != "${src_commit}" ]; then
  echo "Global Install Operator: Failure - Installed operator commit id (${camel_op_commit}) does not match expected commit id (${src_commit})"
  exit 1
fi

echo "Global Operator Install: Success"

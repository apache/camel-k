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
# Perform a cleanup of the test suite
#
####

set -e

while getopts ":b:c:i:x:" opt; do
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
    x)
      SAVE_NAMESPACES=${OPTARG}
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

#
# Reset the proxy to default if using an OLM
# which would require a catalogsource
#
if [ -n "${BUILD_CATALOG_SOURCE_NAMESPACE}" ]; then
  ./.github/actions/kamel-cleanup/reset-proxy.sh
fi

if [ -n "${IMAGE_NAMESPACE}" ]; then
  echo -n "Removing compiled image streams ... "
  imgstreams="camel-k camel-k-bundle camel-k-iib"
  set +e
  for cis in ${imgstreams}
  do
    if kubectl get is ${cis} -n ${IMAGE_NAMESPACE} &> /dev/null
    then
      kubectl delete is ${cis} -n ${IMAGE_NAMESPACE}
    fi
  done
  set -e
  echo "Done"
fi

#
# Remove Catalog Source
#
if [ -n "${BUILD_CATALOG_SOURCE_NAMESPACE}" ]; then
  set +e
  echo -n "Removing testing catalogsource ... "
  kubectl delete catalogsource ${BUILD_CATALOG_SOURCE_NAME} -n ${BUILD_CATALOG_SOURCE_NAMESPACE}
  if [ $? == 0 ]; then
    echo "Done"
  else
    echo "Warning: Catalog Source ${BUILD_CATALOG_SOURCE_NAME} not found in ${BUILD_CATALOG_SOURCE_NAMESPACE}"
  fi

  set -e
fi

#
# Remove installed kamel
#
set +e
if command -v kamel &> /dev/null
then
  kamel uninstall --olm=false --all --skip-crd=false
fi

# Ensure the CRDs are removed
kubectl get crds | grep camel | awk '{print $1}' | xargs kubectl delete crd &> /dev/null

#
# Remove KNative resources
#
./.github/actions/kamel-cleanup/cleanup-knative.sh

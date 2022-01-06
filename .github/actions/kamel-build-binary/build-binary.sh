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
# Builds the kamel binary
#
####

set -e

while getopts ":i:l:m:s:x:" opt; do
  case "${opt}" in
    i)
      IMAGE_NAMESPACE=${OPTARG}
      ;;
    l)
      REGISTRY_PULL_HOST=${OPTARG}
      ;;
    m)
      MAKE_RULES="${OPTARG}"
      ;;
    s)
      REGISTRY_PUSH_HOST=${OPTARG}
      ;;
    x)
      DEBUG_USE_EXISTING_IMAGE=${OPTARG}
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

if [ -n "${REGISTRY_PUSH_HOST}" ]; then
  #
  # Need an image namespace if using a registry
  #
  if [ -z "${IMAGE_NAMESPACE}" ]; then
    echo "Error: image-namespace not defined"
    exit 1
  fi

  #
  # Build with the PUSH host to ensure the correct image:tag
  # for docker to push the image.
  #
  export CUSTOM_IMAGE=${REGISTRY_PUSH_HOST}/${IMAGE_NAMESPACE}/camel-k
fi

if [ -n "${DEBUG_USE_EXISTING_IMAGE}" ] && [ -n "${CUSTOM_IMAGE}" ]; then
  echo "Fetching Kamel from existing build"

  docker pull ${DEBUG_USE_EXISTING_IMAGE}
  id=$(docker create ${DEBUG_USE_EXISTING_IMAGE})
  docker cp $id:/usr/local/bin/kamel .

  docker tag ${DEBUG_USE_EXISTING_IMAGE} ${CUSTOM_IMAGE}:$(make get-version)
  docker push ${CUSTOM_IMAGE}:$(make get-version)
else

  echo "Build Kamel from source"

  RULES="PACKAGE_ARTIFACTS_STRATEGY=download build package-artifacts images"
  if [ -n "${MAKE_RULES}" ]; then
    RULES=" ${MAKE_RULES} "
  fi

  if [ -n "${REGISTRY_PUSH_HOST}" ]; then
    RULES="${RULES} images-push"
  fi

  make ${RULES}
fi

echo "Moving kamel binary to /usr/local/bin"
sudo mv ./kamel /usr/local/bin
echo "Kamel version installed: $(kamel version)"

#
# Use the PULL host to ensure the correct image:tag
# is passed into the tests for the deployment to pull from
#
BUILD_BINARY_LOCAL_IMAGE_NAME="${REGISTRY_PULL_HOST}/${IMAGE_NAMESPACE}/camel-k"
BUILD_BINARY_LOCAL_IMAGE_VERSION="$(make get-version)"
echo "Setting build-binary-local-image-name to ${BUILD_BINARY_LOCAL_IMAGE_NAME}"
echo "::set-output name=build-binary-local-image-name::${BUILD_BINARY_LOCAL_IMAGE_NAME}"
echo "Setting build-binary-local-image-name-version to ${BUILD_BINARY_LOCAL_IMAGE_VERSION}"
echo "::set-output name=build-binary-local-image-version::${BUILD_BINARY_LOCAL_IMAGE_VERSION}"

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
# Builds the kamel bundle image
#
####

set -e

while getopts ":i:l:n:s:v:" opt; do
  case "${opt}" in
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
      REGISTRY_PUSH_HOST=${OPTARG}
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

echo "Build Operator bundle"
if ! command -v kustomize &> /dev/null
then
  echo "kustomize could not be found. Has it not been installed?"
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

if [ -z "${REGISTRY_PUSH_HOST}" ]; then
  echo "Error: image-registry-push-host not defined"
  exit 1
fi

if [ -z "${REGISTRY_PULL_HOST}" ]; then
  echo "Error: image-registry-pull-host not defined"
  exit 1
fi

#
# Using a custom single cluster can allow for use-case that old camel-k images are cached
# (see https://cloud.redhat.com/blog/image-garbage-collection-in-openshift). This is not an
# issue on ephemeral clusters like kind.
# Therefore, need to edit the bundle CSV to ensure the ImagePullPolicy of the camel-k image is
# set to "Always" to mandate that the new image is always pulled.
#
# Use kustomize to patch the deployment resource
#
pushd config/manager > /dev/null
kustomize edit add patch --path patch-image-pull-policy-always.yaml --kind Deployment
popd

#
# Build with the PUSH host to ensure the correct image:tag
# for docker to push the image.
# Note: the CUSTOM_IMAGE must still reference the pull host
# to allow the operator image to be found
#
BUNDLE_IMAGE_BASE_NAME=$(basename $(make get-bundle-image))
export PUSH_BUNDLE_LOCAL_IMAGE=${REGISTRY_PUSH_HOST}/${IMAGE_NAMESPACE}/${BUNDLE_IMAGE_BASE_NAME}
export CUSTOM_IMAGE=${IMAGE_NAME}
export CUSTOM_VERSION=${IMAGE_VERSION}

export PREV_XY_CHANNEL="stable-$(make get-last-released-version | grep -Po '\d+\.\d+')"
echo "PREV_XY_CHANNEL=${PREV_XY_CHANNEL}" >> $GITHUB_ENV
export NEW_XY_CHANNEL=stable-dev-$(make get-version | grep -Po "\d+\.\d+")
echo "NEW_XY_CHANNEL=${NEW_XY_CHANNEL}" >> $GITHUB_ENV

make bundle-push \
  BUNDLE_IMAGE_NAME="${PUSH_BUNDLE_LOCAL_IMAGE}" \
  DEFAULT_CHANNEL="${NEW_XY_CHANNEL}" \
  CHANNELS="${NEW_XY_CHANNEL}"

#
# Use the PULL host to ensure the correct image:tag
# is passed into the tests for the deployment to pull from
#
PULL_BUNDLE_LOCAL_IMAGE="${REGISTRY_PULL_HOST}/${IMAGE_NAMESPACE}/${BUNDLE_IMAGE_BASE_NAME}:${IMAGE_VERSION}"
echo "Setting build-bundle-local-image to ${PULL_BUNDLE_LOCAL_IMAGE}"
echo "::set-output name=build-bundle-local-image::${PULL_BUNDLE_LOCAL_IMAGE}"

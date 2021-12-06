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
# Configures access to the cluster
#
####

set -e

while getopts ":c:k:" opt; do
  case "${opt}" in
    c)
      CLUSTER_CONFIG_DATA=${OPTARG}
      ;;
    k)
      KUBE_CONFIG_DATA=${OPTARG}
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

has_property() {
  if echo "${CLUSTER_CONFIG_DATA}" | grep ${1} &> /dev/null; then
    echo 0
  else
    echo 1
  fi
}

get_property() {
  VAR=$(echo "${CLUSTER_CONFIG_DATA}" | grep ${1})
  echo ${VAR#*=}
}

if [ -z "${KUBE_CONFIG_DATA}" ]; then
  echo "Error: kube config data property cannot be found"
  exit 1
fi

if [ ! $(has_property kube-admin-user-ctx) ]; then
  echo "Error: kube admin context property cannot be found"
  exit 1
fi

if [ ! $(has_property kube-user-ctx) ]; then
  echo "Error: kube user context property cannot be found"
  exit 1
fi

if [ ! $(has_property image-registry-pull-host) ]; then
  echo "Error: image registry pull host property cannot be found"
  exit 1
fi

if [ ! $(has_property image-registry-push-host) ]; then
  echo "Error: image registry build host property cannot be found"
  exit 1
fi

echo "::set-output name=cluster-image-registry-push-host::$(get_property image-registry-push-host)"
echo "::set-output name=cluster-image-registry-pull-host::$(get_property image-registry-pull-host)"
echo "::set-output name=cluster-image-registry-insecure::$(get_property image-registry-insecure)"
echo "::set-output name=cluster-catalog-source-namespace::$(get_property catalog-source-namespace)"

#
# Export the image namespace if defined in the cluster config
#
if [ $(has_property image-namespace) ]; then
  echo "::set-output name=cluster-image-namespace::$(get_property image-namespace)"
fi

#
# Export the context used for admin and user
#
echo "::set-output name=cluster-kube-admin-user-ctx::$(get_property kube-admin-user-ctx)"
echo "::set-output name=cluster-kube-user-ctx::$(get_property kube-user-ctx)"

#
# Keep values private in the log
#
echo "::add-mask::$(get_property image-registry-push-host)"
echo "::add-mask::$(get_property image-registry-pull-host)"
echo "::add-mask::$(get_property kube-admin-user-ctx)"
echo "::add-mask::$(get_property kube-user-ctx)"

#
# Export the flag for olm capability
#
echo "::set-output name=cluster-has-olm::$(get_property has-olm)"

#
# Login to docker if registry is externally secured
#
if [ $(has_property image-registry-user) ] && [ $(has_property image-registry-token) ]; then
  echo "Secured registry in use so login with docker"
  docker login \
    -u $(get_property image-registry-user) \
    -p $(get_property image-registry-token) \
    $(get_property image-registry-push-host)
fi

# Copy the kube config to the correct location for kubectl
mkdir -p $HOME/.kube
echo -n "${KUBE_CONFIG_DATA}" | base64 -d > ${HOME}/.kube/config
if [ ! -f ${HOME}/.kube/config ]; then
  echo "Error: kube config file not created correctly"
  exit 1
fi

set -e
kubectl config use-context $(get_property kube-admin-user-ctx)
if [ $? != 0 ]; then
  echo "Error: Failed to select kube admin context. Is the config and context correct?"
  exit 1
fi
set +e

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
# This script takes care of Knative setup
#
####

KNATIVE_VERSION=1.14.0

TIMEOUT="150s"

set -e

# Get the os/arch
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Prerequisites
#

echo -n "Checking yq ... "
if which yq > /dev/null 2>&1; then
  echo "ok"
else
  echo "not found. (see https://mikefarah.gitbook.io/yq)"
  exit 1
fi

set +e

apply() {
  local file="${1:-}"
  if [ -z "${file}" ]; then
    echo "Error: Cannot apply. No file."
    exit 1
  fi
  echo "Applying ${file} ..."
  kubectl apply --filename ${file}
  if [ $? != 0 ]; then
    sleep 5
    echo "Re-applying ${file} ..."
    kubectl apply --filename ${file}
    if [ $? != 0 ]; then
      echo "Error: Application of resource failed."
      exit 1
    fi
  fi
}

SERVING_VERSION="knative-v${KNATIVE_VERSION}"
EVENTING_VERSION="knative-v${KNATIVE_VERSION}"
KOURIER_VERSION="knative-v${KNATIVE_VERSION}"

SERVING_CRDS="https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-crds.yaml"
SERVING_CORE="https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-core.yaml"
KOURIER="https://github.com/knative-sandbox/net-kourier/releases/download/${KOURIER_VERSION}/kourier.yaml"
EVENTING_CRDS="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-crds.yaml"
EVENTING_CORE="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-core.yaml"
IN_MEMORY_CHANNEL="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/in-memory-channel.yaml"
CHANNEL_BROKER="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/mt-channel-broker.yaml"

KNATIVE_TEMP=$(mktemp -d knative-XXXX)

# Serving CRDs
#

if kubectl get ns knative-serving >/dev/null 2>&1; then
  echo "Knative Serving already installed"
else
  echo "Installing Knative Serving ..."

  orig_yaml="${KNATIVE_TEMP}/serving-crds.yaml"

  curl -L -s ${SERVING_CRDS} > ${orig_yaml}
  if [ -s ${orig_yaml} ]; then
    apply ${orig_yaml}
  else
    echo "Error: Failed to install ${SERVING_CRDS}"
    exit 1
  fi

  # Serving Core
  #

  orig_yaml="${KNATIVE_TEMP}/serving-core.yaml"
  apply_yaml="${KNATIVE_TEMP}/serving-core-apply.yaml"

  curl -L -s ${SERVING_CORE} > ${orig_yaml}
  cat ${orig_yaml} | yq e 'del(.spec.template.spec.containers[].resources)' > ${apply_yaml}
  if [ -s ${apply_yaml} ]; then
    apply ${apply_yaml}
    echo "Waiting for pods to be ready in knative-serving"
    kubectl wait --for=condition=Ready pod --all -n knative-serving --timeout=${TIMEOUT}
  else
    echo "Error: Failed to install ${SERVING_CORE}"
    exit 1
  fi
fi

# Kourier
#

if kubectl get ns kourier-system >/dev/null 2>&1; then
  echo "Kourier already installed"
else
  echo "Installing Kourier ..."

  orig_yaml="${KNATIVE_TEMP}/kourier.yaml"

  curl -L -s ${KOURIER} > ${orig_yaml}
  if [ -s ${orig_yaml} ]; then
    apply ${orig_yaml}
  else
    echo "Error: Failed to install ${KOURIER}"
    exit 1
  fi

  sleep 5

  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
  if [ $? != 0 ]; then
    echo "Error: Failed to patch configmap"
    exit 1
  fi
fi

# Eventing CRDs
#

if kubectl get ns knative-eventing >/dev/null 2>&1; then
  echo "Knative Eventing already installed"
else
  echo "Installing Knative Eventing ..."

  orig_yaml="${KNATIVE_TEMP}/eventing-crds.yaml"

  curl -L -s ${EVENTING_CRDS} > ${orig_yaml}
  if [ -s ${orig_yaml} ]; then
    apply ${orig_yaml}
  else
    echo "Error: Failed to install ${EVENTING_CRDS}"
    exit 1
  fi

  # Eventing Core
  #

  orig_yaml="${KNATIVE_TEMP}/eventing-core.yaml"
  apply_yaml="${KNATIVE_TEMP}/eventing-core-apply.yaml"

  curl -L -s ${EVENTING_CORE} > ${orig_yaml}
  cat ${orig_yaml} | yq e 'del(.spec.template.spec.containers[].resources)' > ${apply_yaml}
  if [ -s ${apply_yaml} ]; then
    apply ${apply_yaml}
    echo "Waiting for pods to be ready in knative-eventing"
    kubectl wait --for=condition=Ready pod --all -n knative-eventing --timeout=${TIMEOUT}
  else
    echo "Error: Failed to install ${EVENTING_CORE}"
    exit 1
  fi

  # Eventing Channels

  orig_yaml="${KNATIVE_TEMP}/in-memory-channel.yaml"
  apply_yaml="${KNATIVE_TEMP}/in-memory-channel-apply.yaml"

  curl -L -s ${IN_MEMORY_CHANNEL} > ${orig_yaml}
  cat ${orig_yaml} | yq e 'del(.spec.template.spec.containers[].resources)' > ${apply_yaml}
  if [ -s ${apply_yaml} ]; then
    apply ${apply_yaml}
  else
    echo "Error: Failed to install ${IN_MEMORY_CHANNEL}"
    exit 1
  fi

  # Eventing Broker
  #

  orig_yaml="${KNATIVE_TEMP}/channel-broker.yaml"
  apply_yaml="${KNATIVE_TEMP}/channel-broker-apply.yaml"

  curl -L -s ${CHANNEL_BROKER} > ${orig_yaml}
  cat ${orig_yaml} | yq e 'del(.spec.template.spec.containers[].resources)' > ${apply_yaml}
  if [ -s ${apply_yaml} ]; then
    apply ${apply_yaml}
  else
    echo "Error: Failed to install ${CHANNEL_BROKER}"
    exit 1
  fi

  # Eventing sugar controller configuration
  echo "Patching Knative eventing configuration"
  kubectl patch configmap/config-sugar \
    -n knative-eventing \
    --type merge \
    -p '{"data":{"namespace-selector":"{\"matchExpressions\":[{\"key\":\"eventing.knative.dev/injection\",\"operator\":\"In\",\"values\":[\"enabled\"]}]}"}}'

  kubectl patch configmap/config-sugar \
    -n knative-eventing \
    --type merge \
    -p '{"data":{"trigger-selector":"{\"matchExpressions\":[{\"key\":\"eventing.knative.dev/injection\",\"operator\":\"In\",\"values\":[\"enabled\"]}]}"}}'
fi

# Wait for installation completed
echo && echo "Waiting for all pods to be ready in knative-serving"
kubectl wait --for=condition=Ready pod --all -n knative-serving --timeout=${TIMEOUT}
echo && echo "Waiting for all pods to be ready in kourier-system"
kubectl wait --for=condition=Ready pod --all -n kourier-system --timeout=${TIMEOUT}
echo && echo "Waiting for all pods to be ready in knative-eventing"
kubectl wait --for=condition=Ready pod --all -n knative-eventing --timeout=${TIMEOUT}

# Expose Kourier Service
#
if [[ "${OS}" = "darwin" ]]; then
  echo
  kubectl get svc -n kourier-system
  echo
  echo "On ${OS} you may want to run 'minikube tunnel' to expose the Kourier Service"
fi

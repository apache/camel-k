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
# Install the knative setup
#
####

set -e

# Prerequisites
sudo wget https://github.com/mikefarah/yq/releases/download/v4.26.1/yq_linux_amd64 -O /usr/bin/yq && sudo chmod +x /usr/bin/yq

set +e

export SERVING_VERSION=knative-v1.3.0
export EVENTING_VERSION=knative-v1.3.3
export KOURIER_VERSION=knative-v1.3.0

apply() {
  local file="${1:-}"
  if [ -z "${file}" ]; then
    echo "Error: Cannot apply. No file."
    exit 1
  fi

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

SERVING_CRDS="https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-crds.yaml"
SERVING_CORE="https://github.com/knative/serving/releases/download/${SERVING_VERSION}/serving-core.yaml"
KOURIER="https://github.com/knative/net-kourier/releases/download/${KOURIER_VERSION}/kourier.yaml"
EVENTING_CRDS="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-crds.yaml"
EVENTING_CORE="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-core.yaml"
IN_MEMORY_CHANNEL="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/in-memory-channel.yaml"
CHANNEL_BROKER="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/mt-channel-broker.yaml"
SUGAR_CONTROLLER="https://github.com/knative/eventing/releases/download/${EVENTING_VERSION}/eventing-sugar-controller.yaml"

# Serving
apply "${SERVING_CRDS}"

YAML=$(mktemp serving-core-XXX.yaml)
curl -L -s ${SERVING_CORE} | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - > ${YAML}
if [ -s ${YAML} ]; then
  apply ${YAML}
  echo "Waiting for pods to be ready in knative-serving (dependency for kourier)"
  kubectl wait --for=condition=Ready pod --all -n knative-serving --timeout=60s
else
  echo "Error: Failed to correctly download ${SERVING_CORE}"
  exit 1
fi

# Kourier
apply "${KOURIER}"

sleep 5

kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
if [ $? != 0 ]; then
  echo "Error: Failed to patch configmap"
  exit 1
fi

# Eventing
apply "${EVENTING_CRDS}"

YAML=$(mktemp eventing-XXX.yaml)
curl -L -s ${EVENTING_CORE} | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - > ${YAML}
if [ -s ${YAML} ]; then
  apply ${YAML}
else
  echo "Error: Failed to correctly download ${SERVING_CORE}"
  exit 1
fi

# Eventing channels
YAML=$(mktemp in-memory-XXX.yaml)
curl -L -s ${IN_MEMORY_CHANNEL} | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - > ${YAML}
if [ -s ${YAML} ]; then
  apply ${YAML}
else
  echo "Error: Failed to correctly download ${SERVING_CORE}"
  exit 1
fi

# Eventing broker
YAML=$(mktemp channel-broker-XXX.yaml)
curl -L -s ${CHANNEL_BROKER} | head -n -1 | yq e 'del(.spec.template.spec.containers[].resources)' - > ${YAML}
if [ -s ${YAML} ]; then
  apply ${YAML}
else
  echo "Error: Failed to correctly download ${SERVING_CORE}"
  exit 1
fi

# Eventing sugar controller for injection
apply ${SUGAR_CONTROLLER}

# Wait for installation completed
echo "Waiting for all pods to be ready in kourier-system"
kubectl wait --for=condition=Ready pod --all -n kourier-system --timeout=60s
echo "Waiting for all pods to be ready in knative-serving"
kubectl wait --for=condition=Ready pod --all -n knative-serving --timeout=60s
echo "Waiting for all pods to be ready in knative-eventing"
kubectl wait --for=condition=Ready pod --all -n knative-eventing --timeout=60s

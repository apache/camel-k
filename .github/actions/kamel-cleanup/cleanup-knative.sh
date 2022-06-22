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
# Perform a cleanup of knative installation
#
####
set +e

cleanup_resources() {
  local kind="${1}"
  local needle="${2}"

  # Stops "Resource not found message" and exit code to 0
  local result=$(kubectl --ignore-not-found=true get ${kind} | grep ${needle})
  if [ $? == 0 ]; then
    names=$(echo "${result}" | awk '{print $1}')
    for r in ${names}
    do
      # Timeout after 10 minutes
      kubectl delete --now --timeout=600s ${kind} ${r} 1> /dev/null
    done
  fi

  echo "Done"
}

# Remove any namespaces
echo -n "Removing testing knative namespaces ... "
cleanup_resources ns 'knative-eventing\|knative-serving\|kourier-system'

# Mutating webhooks
echo -n "Removing testing knative mutating webhooks ... "
cleanup_resources MutatingWebhookConfiguration 'knative.dev'

# Validating webhooks
echo -n "Removing testing knative validating webhooks ... "
cleanup_resources ValidatingWebhookConfiguration 'knative.dev'

# Cluster Role Bindings
echo -n "Removing testing knative cluster role bindings ... "
cleanup_resources clusterrolebindings 'kourier\|knative\|imc'

# Cluster Roles
echo -n "Removing testing knative cluster roles ... "
cleanup_resources clusterroles 'knative\|imc'

# CRDS
CRDS=$(kubectl --ignore-not-found=true get crds | grep knative | awk '{print $1}')
if [ -n "${CRDS}" ]; then
  for crd in ${CRDS}
  do
    echo -n "Removing ${crd} CRD ... "
    cleanup_resources crds "${crd}"
  done
fi
set -e

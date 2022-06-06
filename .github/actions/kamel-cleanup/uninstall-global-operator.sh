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
# Execute an uninstall of a global operator
#
####

set +e

while getopts ":c:g:" opt; do
  case "${opt}" in
    c)
      BUILD_CATALOG_SOURCE_NAMESPACE=${OPTARG}
      ;;
    g)
      GLOBAL_OPERATOR_NAMESPACE=${OPTARG}
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

if [ -z "${GLOBAL_OPERATOR_NAMESPACE}" ]; then
  # Not defined so no need to go any further
  exit 0
fi

#
# Determine if olm was used based on catalog source definition
#
has_olm="false"
if [ -n "${BUILD_CATALOG_SOURCE_NAMESPACE}" ]; then
  has_olm="true"
fi

#
# Check the global namespace
#
kubectl get ns ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null
if [ $? != 0 ]; then
  echo "Info: The ${GLOBAL_OPERATOR_NAMESPACE} namespace does not exist. Nothing to do."
  exit 0
fi

kubectl get deploy camel-k-operator -n ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null
if [ $? != 0 ]; then
  echo "Info: The operator is not installed in the global operator namespace (${GLOBAL_OPERATOR_NAMESPACE}). Nothing to do."
  exit 0
fi

#
# Uninstall the operator
# (ignore errors if nothing is installed)
#
set +e

if [ "${has_olm}" == "true" ]; then
  # Delete subscription if present
  kubectl get subs -n ${GLOBAL_OPERATOR_NAMESPACE} | \
    grep camel-k | awk '{print $1}' | \
    xargs -I '{}' kubectl delete subs '{}' -n ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null

  # Delete CSV if present
  kubectl get csv -n ${GLOBAL_OPERATOR_NAMESPACE} | \
    grep camel-k | awk '{print $1}' | \
    xargs -I '{}' kubectl delete csv '{}' -n ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null
else
  kubectl get deploy -n ${GLOBAL_OPERATOR_NAMESPACE} | \
    grep camel-k | awk '{print $1}' | \
    xargs -I '{}' kubectl delete deploy '{}' -n ${GLOBAL_OPERATOR_NAMESPACE} &> /dev/null
fi

sleep 3

#
# Wait for the operator to be removed
#
timeout=180
i=1
command="kubectl get pods -n ${GLOBAL_OPERATOR_NAMESPACE} 2> /dev/null | grep camel-k &> /dev/null"

while eval "${command}"
do
  ((i++))
  if [ "${i}" -gt "${timeout}" ]; then
    echo "kamel operator not successfully uninstalled, aborting due to ${timeout}s timeout"
    exit 1
  fi

  sleep 1
done

echo "Camel K operator uninstalled."

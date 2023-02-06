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
# Install the kamel setup using the admin context
#
####

set -e

while getopts ":a:" opt; do
  case "${opt}" in
    a)
      KUBE_ADMIN_CTX=${OPTARG}
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

if [ -z "${KUBE_ADMIN_CTX}" ]; then
  echo "Error: kube-admin-user-ctx not defined"
  exit 1
fi

#
# Get current context
#
ctx=$(kubectl config current-context)

#
# Need to be admin so switch to the admin context
#
kubectl config use-context "${KUBE_ADMIN_CTX}"

#
# Ensure built binary CRDs are always installed by turning off olm
#
kamel install --cluster-setup --olm=false

#
# Change back to original context
#
kubectl config use-context "${ctx}"

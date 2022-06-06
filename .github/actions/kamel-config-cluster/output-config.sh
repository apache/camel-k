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
# Outputs the config to cluster output variables
#
####

set -e

while getopts ":a:b:c:g:n:o:p:q:s:u:" opt; do
  case "${opt}" in
    a)
      ADMIN_USER_CTX=${OPTARG}
      ;;
    b)
      CATALOG_SOURCE_NAME=${OPTARG}
      ;;
    c)
      CATALOG_SOURCE_NAMESPACE=${OPTARG}
      ;;
    g)
      GLOBAL_OPERATOR_NAMESPACE=${OPTARG}
      ;;
    n)
      IMAGE_NAMESPACE=${OPTARG}
      ;;
    o)
      HAS_OLM=${OPTARG}
      ;;
    p)
      PUSH_HOST=${OPTARG}
      ;;
    q)
      PULL_HOST=${OPTARG}
      ;;
    s)
      INSECURE=${OPTARG}
      ;;
    u)
      USER_CTX=${OPTARG}
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

echo "::set-output name=cluster-image-registry-push-host::${PUSH_HOST}"
echo "::set-output name=cluster-image-registry-pull-host::${PULL_HOST}"
echo "::set-output name=cluster-image-registry-insecure::${INSECURE}"
echo "::set-output name=cluster-kube-admin-user-ctx::${ADMIN_USER_CTX}"
echo "::set-output name=cluster-kube-user-ctx::${USER_CTX}"

# Set the image namespace
echo "::set-output name=cluster-image-namespace::${IMAGE_NAMESPACE}"

# Set the catalog source
echo "::set-output name=cluster-catalog-source-name::${CATALOG_SOURCE_NAME}"
echo "::set-output name=cluster-catalog-source-namespace::${CATALOG_SOURCE_NAMESPACE}"

#
# Export the flag for olm capability
#
echo "::set-output name=cluster-has-olm::${HAS_OLM}"

#
# Export the flag for testing using global operator
#
echo "::set-output name=cluster-global-operator-namespace::${GLOBAL_OPERATOR_NAMESPACE}"

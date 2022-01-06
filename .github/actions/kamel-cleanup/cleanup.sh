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
# Perform a cleanup of the test suite
#
####

set -e

while getopts ":c:" opt; do
  case "${opt}" in
    c)
      BUILD_CATALOG_SOURCE=${OPTARG}
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
# Remove installed kamel
#
set +e
if command -v kamel &> /dev/null
then
  kamel uninstall --olm=false --all
fi

# Ensure the CRDs are removed
kubectl get crds | grep camel | awk '{print $1}' | xargs kubectl delete crd &> /dev/null
set -e

#
# Remove Catalog Source
#
if [ -z "${BUILD_CATALOG_SOURCE}" ]; then
  # Catalog source never defined so nothing to do
  exit 0
fi

set +e
CATALOG_NS=$(kubectl get catalogsource --all-namespaces | grep ${BUILD_CATALOG_SOURCE} | awk {'print $1'})
for ns in ${CATALOG_NS}
do
  kubectl delete CatalogSource ${BUILD_CATALOG_SOURCE} -n ${ns}
done
set -e

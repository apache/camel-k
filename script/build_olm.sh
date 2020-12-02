#!/bin/sh

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

echo "=== This is the deprecated generation of csv metadata. Use 'make bundle' for the newer bundle format ==="

location=$(dirname $0)
olm_catalog=${location}/../deploy/olm-catalog
config=${location}/../config

cd $location/..

version=$(make -s get-version | tr '[:upper:]' '[:lower:]')
olm_dest=${olm_catalog}/camel-k-dev

if [ -d ${olm_dest}/triage ]; then
  rm -rf ${olm_dest}/triage
fi

#
# Use the triage directory for building the csv to avoid
# overwriting the CRDs and needlessly changing their format
#
mkdir -p ${olm_dest}/triage

kustomize build config/manifests | operator-sdk generate packagemanifests \
  --version ${version} \
  --output-dir ${olm_dest}/triage \
  --channel stable \
  --default-channel

if [ -f ${olm_dest}/triage/${version}/camel-k.clusterserviceversion.yaml ]; then
  cp -f ${olm_dest}/triage/${version}/camel-k.clusterserviceversion.yaml ${olm_dest}/${version}/camel-k.v${version}.clusterserviceversion.yaml
else
  echo "ERROR: Failed to generate the CSV package manifest"
  exit 1
fi

if [ ${olm_dest}/triage/camel-k.package.yaml ]; then
  cp -f ${olm_dest}/triage/camel-k.package.yaml ${olm_dest}/camel-k-dev.package.yaml
fi

if [ -d ${olm_dest}/triage ]; then
  rm -rf ${olm_dest}/triage
fi

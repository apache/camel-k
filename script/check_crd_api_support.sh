#!/bin/bash

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

is_binary_available() {

  client="${1}"

  # Check path first if it already exists
  set +e
  which "${client}" &>/dev/null
  if [ $? -eq 0 ]; then
    set -e
    echo "OK"
    return
  fi

  set -e

  # Error, no oc found
  echo "ERROR: No '${client}' binary found in path."
}

location=$(dirname $0)
rootdir=$location/../

cd $rootdir

client="oc"
hasclient=$(is_binary_available "${client}")
if [ "${hasclient}" != "OK" ]; then
  client="kubectl"
	hasclient=$(is_binary_available "${client}")
	if [ "${hasclient}" != "OK" ]; then
	  echo "ERROR: No kube client installed."
	  exit 1
	fi
fi

crd_version=$("${client}" explain customresourcedefinitions | grep VERSION | awk '{print $2}')
api="apiextensions.k8s.io"

if [ "${crd_version}" == "${api}/v1beta1" ]; then
	echo "ERROR: CRD API version is too old to install camel-k in this way. Try using the client CLI app, which is able to convert the APIs."
elif [ "${crd_version}" != "${api}/v1" ]; then
	echo "ERROR: CRD API version '${crd_version}' is not supported."
else
	echo "OK"
fi

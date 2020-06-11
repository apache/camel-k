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

# Exit on error
set -e

eval $(crc oc-env)
user=$(oc whoami)
if [ "$project" = "" ]; then
  project=$(oc project -q)
else
  oc new-project $project 2>/dev/null || true
fi

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

make images-dev

# Tag image
docker tag apache/camel-k:$1 default-route-openshift-image-registry.apps-crc.testing/$project/camel-k:$1
# Login to Docker registry
if [ $user = "kube:admin" ]; then
  docker login -u kubeadmin -p $(oc whoami -t) default-route-openshift-image-registry.apps-crc.testing
else 
  docker login -u $(oc whoami) -p $(oc whoami -t) default-route-openshift-image-registry.apps-crc.testing
fi
# Push image to Docker registry
docker push default-route-openshift-image-registry.apps-crc.testing/$project/camel-k:$1

# Try setup with standard user
ret=0
cmd="./kamel install --olm=false --operator-image=image-registry.openshift-image-registry.svc:5000/$project/camel-k:$1"
eval "$cmd -n $project 2>/dev/null || export ret=\$?"

if [ $ret -ne 0 ]; then
  # Login as admin if cluster setup fails with standard user
  oc login -u kubeadmin
  eval "$cmd --cluster-setup"
  oc login -u $user
  eval "$cmd -n $project"
fi

oc delete pod -l name=camel-k-operator -n $project || true

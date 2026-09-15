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

CERT_MANAGER_VERSION="v1.21.2"
TIMEOUT="150s"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml
kubectl wait --for=condition=available deployment/cert-manager -n cert-manager --timeout=$TIMEOUT
kubectl wait --for=condition=available deployment/cert-manager-webhook -n cert-manager --timeout=$TIMEOUT
kubectl wait --for=condition=available deployment/cert-manager-cainjector -n cert-manager --timeout=$TIMEOUT

# Install a self-signed ClusterIssuer for test purposes
kubectl apply -f $SCRIPT_DIR/cluster-issuer.yaml

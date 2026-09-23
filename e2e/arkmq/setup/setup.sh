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

####
#
# This script takes care of ArkMQ operator setup for e2e tests
#
####

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl create namespace arkmq --dry-run=client -o yaml | kubectl apply -f -
kubectl apply --server-side -f https://github.com/arkmq-org/arkmq-org-broker-operator/releases/latest/download/arkmq-org-broker-operator.yaml -n arkmq
kubectl rollout status deployment arkmq-org-broker-operator -n arkmq --timeout=180s

# Wait for CRDs to be established
kubectl wait --for=condition=established crd/activemqartemises.broker.amq.io --timeout=60s
kubectl wait --for=condition=established crd/activemqartemisaddresses.broker.amq.io --timeout=60s

#### Setup an ActiveMQ Artemis broker
kubectl apply -f $SCRIPT_DIR/broker.yaml -n arkmq
kubectl wait activemqartemis/my-broker --for=condition=Ready --timeout=300s -n arkmq

#### Setup an ActiveMQ Artemis queue address
kubectl apply -f $SCRIPT_DIR/queue.yaml -n arkmq

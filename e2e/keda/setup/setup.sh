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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#### 1. Setup KEDA Operator
# Installs KEDA operator into 'keda' namespace
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.17.2/keda-2.17.2.yaml
kubectl rollout status deployment keda-operator -n keda --timeout=180s

#### 2. Setup Plain Ephemeral Kafka (No Operator required)
kubectl apply -f $SCRIPT_DIR/kafka-ephemeral.yaml -n keda
kubectl rollout status deployment my-cluster-kafka -n keda --timeout=180s
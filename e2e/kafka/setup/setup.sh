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
# This script takes care of Strimzi setup as described in https://strimzi.io/quickstarts/
#
####

kubectl create namespace kafka
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
kubectl rollout status deployment strimzi-cluster-operator -n kafka --timeout=180s

#### Setup a Kafka cluster which we'll use for testing
kubectl apply -f ./e2e/kafka/setup/kafka-ephemeral.yaml
kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka

#### Setup a Kafka topic which we'll use for testing
kubectl apply -f ./e2e/kafka/setup/kafka-topic.yaml
kubectl wait kafkatopic/my-topic --for=condition=Ready --timeout=60s -n kafka

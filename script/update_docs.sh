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

location=$(dirname $0)

echo "Scraping information from go.mod"
KNATIVE_API_VERSION=$(grep '^.*knative.dev/eventing ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
KUBE_API_VERSION=$(grep '^.*k8s.io/api ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
OPERATOR_FWK_API_VERSION=$(grep '^.*github.com/operator-framework/api ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
SERVICE_BINDING_OP_VERSION=$(grep '^.*github.com/redhat-developer/service-binding-operator ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')
PROMETHEUS_OP_VERSION=$(grep '^.*github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring ' $location/../go.mod | sed 's/^.* //' | sed 's/^.//')

echo "Kubernetes API version: $KUBE_API_VERSION"
echo "Operator Framework API version: $OPERATOR_FWK_API_VERSION"
echo "Knative API version: $KNATIVE_API_VERSION"
echo "Service Binding Operator version: $SERVICE_BINDING_OP_VERSION"
echo "Prometheus Operator version: $PROMETHEUS_OP_VERSION"

yq -i ".asciidoc.attributes.kubernetes-api-version = \"$KUBE_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.operator-fwk-api-version = \"$OPERATOR_FWK_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.knative-api-version = \"$KNATIVE_API_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.service-binding-op-version = \"$SERVICE_BINDING_OP_VERSION\"" $location/../docs/antora.yml
yq -i ".asciidoc.attributes.prometheus-op-version = \"$PROMETHEUS_OP_VERSION\"" $location/../docs/antora.yml

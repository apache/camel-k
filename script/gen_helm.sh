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

set -e

location=$(dirname "$0")

cd $location/../

cat ./script/headers/yaml.txt > ./helm/camel-k/templates/rbacs-common.yaml
cat ./script/headers/yaml.txt > ./helm/camel-k/templates/rbacs-descoped.yaml
cat ./script/headers/yaml.txt > ./helm/camel-k/templates/rbacs-namespaced.yaml

kustomize build ./pkg/resources/config/helm/common/. >> ./helm/camel-k/templates/rbacs-common.yaml

printf "{{- if eq .Values.operator.global \"false\" }}\n" >> ./helm/camel-k/templates/rbacs-namespaced.yaml
kustomize build ./pkg/resources/config/helm/namespaced/. >> ./helm/camel-k/templates/rbacs-namespaced.yaml
printf "{{- end }}\n" >> ./helm/camel-k/templates/rbacs-namespaced.yaml

printf "{{- if eq .Values.operator.global \"true\" }}\n" >> ./helm/camel-k/templates/rbacs-descoped.yaml
kustomize build ./pkg/resources/config/helm/descoped/. >> ./helm/camel-k/templates/rbacs-descoped.yaml
printf "{{- end }}\n" >> ./helm/camel-k/templates/rbacs-descoped.yaml

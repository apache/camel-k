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

location=$(dirname $0)

GO111MODULE=on

# Entering the client module
cd $location/../pkg/client/camel

echo "Generating Go client code..."

$(go env GOPATH)/bin/applyconfiguration-gen \
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1" \
	--go-header-file=../../../script/headers/default.txt \
	--output-dir=./applyconfiguration/ \
	--output-pkg=github.com/apache/camel-k/v2/pkg/client/camel/applyconfiguration

$(go env GOPATH)/bin/client-gen \
	--input camel/v1 \
	--go-header-file=../../../script/headers/default.txt \
	--clientset-name "versioned"  \
	--input-base=github.com/apache/camel-k/v2/pkg/apis \
	--apply-configuration-package=github.com/apache/camel-k/v2/pkg/client/camel/applyconfiguration \
	--output-dir=./clientset/ \
	--output-pkg=github.com/apache/camel-k/v2/pkg/client/camel/clientset

$(go env GOPATH)/bin/client-gen \
  --input strimzi/v1beta2 \
  --go-header-file=../../../script/headers/default.txt \
  --input-base=github.com/apache/camel-k/v2/pkg/apis/duck \
  --output-dir=../clientset/ \
  --output-pkg=github.com/apache/camel-k/v2/pkg/client/duck/strimzi/clientset

$(go env GOPATH)/bin/lister-gen \
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1" \
	--go-header-file=../../../script/headers/default.txt \
	--output-dir=./listers/ \
	--output-pkg=github.com/apache/camel-k/v2/pkg/client/camel/listers

$(go env GOPATH)/bin/informer-gen \
    "github.com/apache/camel-k/v2/pkg/apis/camel/v1" \
	--versioned-clientset-package=github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned \
	--listers-package=github.com/apache/camel-k/v2/pkg/client/camel/listers \
	--go-header-file=../../../script/headers/default.txt \
	--output-dir=./informers/ \
	--output-pkg=github.com/apache/camel-k/v2/pkg/client/camel/informers

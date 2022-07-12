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

unset GOPATH
GO111MODULE=on

# Entering the client module
cd $location/../pkg/client/camel

echo "Generating Go client code..."

$(go env GOPATH)/bin/client-gen \
	--input=camel/v1,camel/v1alpha1 \
	--go-header-file=../../../script/headers/default.txt \
	--clientset-name "versioned"  \
	--input-base=github.com/apache/camel-k/pkg/apis \
	--output-base=. \
	--output-package=github.com/apache/camel-k/pkg/client/camel/clientset

$(go env GOPATH)/bin/lister-gen \
	--input-dirs=github.com/apache/camel-k/pkg/apis/camel/v1,github.com/apache/camel-k/pkg/apis/camel/v1alpha1 \
	--go-header-file=../../../script/headers/default.txt \
	--output-base=. \
	--output-package=github.com/apache/camel-k/pkg/client/camel/listers

$(go env GOPATH)/bin/informer-gen \
    --versioned-clientset-package=github.com/apache/camel-k/pkg/client/camel/clientset/versioned \
	--listers-package=github.com/apache/camel-k/pkg/client/camel/listers \
	--input-dirs=github.com/apache/camel-k/pkg/apis/camel/v1,github.com/apache/camel-k/pkg/apis/camel/v1alpha1 \
	--go-header-file=../../../script/headers/default.txt \
	--output-base=. \
	--output-package=github.com/apache/camel-k/pkg/client/camel/informers


# hack to fix non go-module compliance
rm -rf ./clientset
rm -rf ./informers
rm -rf ./listers
cp -R ./github.com/apache/camel-k/pkg/client/camel/* .
rm -rf ./github.com

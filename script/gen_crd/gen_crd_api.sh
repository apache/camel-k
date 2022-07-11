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
rootdir=$location/../..
crd_file_camel=$rootdir/docs/modules/ROOT/partials/apis/camel-k-crds.adoc
crd_file_kamelets=$rootdir/docs/modules/ROOT/partials/apis/kamelets-crds.adoc

# Until the pull req below is merged upstream, we need to use a self-hosted
# version of gen-crd-api-reference-docs:
#   https://github.com/ahmetb/gen-crd-api-reference-docs/pull/45

echo "Generating CRD API documentation..."
# to run a local copy use something like
#go run /Users/david/projects/camel/gen-crd-api-reference-docs/main.go \
#you will probably need to comment out use of blackfriday.
go run github.com/tadayosi/gen-crd-api-reference-docs@v0.4.0-camel-k-1 \
    -config $location/gen-crd-api-config.json \
    -template-dir $location/template \
    -api-dir "github.com/apache/camel-k/pkg/apis/camel/v1" \
    -out-file $crd_file_camel

#go run /Users/david/projects/camel/gen-crd-api-reference-docs/main.go \
go run github.com/tadayosi/gen-crd-api-reference-docs@v0.4.0-camel-k-1 \
    -config $location/gen-kamelets-crd-api-config.json \
    -template-dir $location/template \
    -api-dir "github.com/apache/camel-k/pkg/apis/camel/v1alpha1" \
    -out-file $crd_file_kamelets

echo "Generating CRD API documentation... Done."

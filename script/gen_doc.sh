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
rootdir=$location/..

echo "Generating API documentation..."
$location/gen_crd/gen_crd_api.sh
echo "Generating API documentation... done!"

echo "Generating traits documentation..."
cd $rootdir
go run ./cmd/util/doc-gen \
  --input-dirs github.com/apache/camel-k/pkg/apis/camel/v1/trait \
  --input-dirs github.com/apache/camel-k/addons/keda \
  --input-dirs github.com/apache/camel-k/addons/master \
  --input-dirs github.com/apache/camel-k/addons/threescale \
  --input-dirs github.com/apache/camel-k/addons/tracing
echo "Generating traits documentation... done!"

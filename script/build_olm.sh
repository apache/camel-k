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
olm_catalog=${location}/../deploy/olm-catalog

if [ "$#" -ne 1 ]; then
    echo "usage: $0 version"
    exit 1
fi

version=$1

cd $location/..

operator-sdk generate csv --csv-version ${version} --csv-config deploy/olm-catalog/csv-config.yaml --update-crds

rm $olm_catalog/camel-k/${version}/crd-*.yaml 2>/dev/null || true
cp $location/../deploy/crd-*.yaml $olm_catalog/camel-k/${version}/

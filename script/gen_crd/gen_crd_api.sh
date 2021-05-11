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

echo "Downloading gen-crd-api-reference-docs binary..."
TMPFILE=`mktemp`
TMPDIR=`mktemp -d`
PWD=`pwd`
wget -q --show-progress https://github.com/ahmetb/gen-crd-api-reference-docs/releases/download/v0.1.5/gen-crd-api-reference-docs_linux_amd64.tar.gz -O $TMPFILE
tar -C $TMPDIR -xf $TMPFILE

echo "Generating CRD API documentation..."
$TMPDIR/gen-crd-api-reference-docs \
    -config $location/gen-crd-api-config.json \
    -template-dir $location/template \
    -api-dir "github.com/apache/camel-k/pkg/apis/camel" \
    -out-file $rootdir/docs/modules/ROOT/pages/apis/crds-html.adoc

# Workaround: https://github.com/ahmetb/gen-crd-api-reference-docs/issues/33
sed -i -E "s/%2f/\//" $rootdir/docs/modules/ROOT/pages/apis/crds-html.adoc

echo "Cleaning the gen-crd-api-reference-docs binary..."
rm $TMPFILE
rm -rf $TMPDIR
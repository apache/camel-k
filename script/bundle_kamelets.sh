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
rootdir=$location/../

set -e

repo=$1
tag=$2

cd $rootdir
target=./build/_kamelets

# Always recreate the dir
rm -rf $target
mkdir $target

if [ "$repo" = "" ]; then
	echo "no kamelet catalog defined: skipping"
	exit 0
fi

if [ "$tag" = "" ]; then
	echo "ERROR: no kamelet catalog version defined"
	exit 1
fi

echo "Cloning repository $repo from tag $tag to bundle kamelets..."

rm -rf ./tmp_kamelet_catalog
git clone -q -c advice.detachedHead=false -b $tag --single-branch --depth 1 $repo ./tmp_kamelet_catalog

cp ./tmp_kamelet_catalog/kamelets/*.kamelet.yaml $target

rm -rf ./tmp_kamelet_catalog

#
# Check that all the kamelets have licences
#
./script/add_license.sh $target ./script/headers/yaml.txt

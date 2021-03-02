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

set -e

repo=$1
branch=$2

cd $location/../
target=./deploy/kamelets
rm -rf $target

if [ "$repo" = "" ]; then
	echo "no kamelet catalog defined: skipping"
	exit 0
fi

if [ "$branch" = "" ]; then
  branch="master"
fi

echo "Cloning repository $repo on branch $branch to bundle kamelets..."


rm -rf ./tmp_kamelet_catalog
git clone -b $branch --single-branch --depth 1 $repo ./tmp_kamelet_catalog

mkdir $target
cp ./tmp_kamelet_catalog/*.kamelet.yaml $target

echo "This directory has been auto-generated from the $branch branch of the $repo repository: do not edit the contained files (changes will be overwritten)" > $target/auto-generated.txt

rm -rf ./tmp_kamelet_catalog

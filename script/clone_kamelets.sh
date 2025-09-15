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

# Entering the api module
cd $location/../pkg/apis/camel/v1

echo "Cloning Kamelets from apache-kamelets repository..."

# TODO: use a fixed tag instad of main starting from apache kamelets 4.15.0 release
wget -q -O kamelet_types.go  https://raw.githubusercontent.com/apache/camel-kamelets/refs/heads/main/crds/pkg/apis/camel/v1/kamelet_types.go
wget -q -O kamelet_types_support.go  https://raw.githubusercontent.com/apache/camel-kamelets/refs/heads/main/crds/pkg/apis/camel/v1/kamelet_types_support.go

# Add a short autogen comment here
comment="// DO NOT EDIT: this file was automatically copied from apache/camel-kamelets/crds project"
sed -i "/^package v1/i $comment" kamelet_types.go
sed -i "/^package v1/i $comment" kamelet_types_support.go
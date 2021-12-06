#!/bin/bash

# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

####
#
# Outputs the kind config to output variables
#
####

set -e

# Kind has the same interface for both pushing and pulling images in its registry
echo "::set-output name=cluster-image-registry-push-host::${KIND_REGISTRY}"
echo "::set-output name=cluster-image-registry-pull-host::${KIND_REGISTRY}"
echo "::set-output name=cluster-image-registry-insecure::$(echo true)"

#
# Export the context used for admin and user
# Since kind has no rbac switched on then these can be the same
#
echo "::set-output name=cluster-kube-admin-user-ctx::$(kubectl config current-context)"
echo "::set-output name=cluster-kube-user-ctx::$(kubectl config current-context)"

# Set the image namespace
echo "::set-output name=cluster-image-namespace::$(echo apache)"

#
# cluster-catalog-source-namespace intentionally blank as OLM not routinely installed
# upgrade tests will install their own catalog-source
#

#
# Export the flag for olm capability
#
echo "::set-output name=cluster-has-olm::$(echo false)"

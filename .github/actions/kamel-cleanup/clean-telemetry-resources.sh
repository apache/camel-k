#!/bin/bash

# ---------------------------------------------------------------------------
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
# ---------------------------------------------------------------------------


#
# Remove any telemetry groups resources that might have been deployed for tests.
# All the telemetry resources are deployed in otlp namespace.
#

set +e

#
# Find if the namespace containing telemetry resources exists
#
namespace_otlp=$(kubectl --ignore-not-found=true get namespaces otlp)
if [ -z "${namespace_otlp}" ]; then
  echo "No telemetry resource installed"
  exit 0
fi

echo "Telemetry namespace exists: ${namespace_otlp}"

#
# Delete telemetry resources namespace
#
kubectl delete --now --timeout=600s namespace ${namespace_otlp} 1> /dev/null

echo "Telemetry resources deleted"
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

set +e

kubectl get Proxy cluster >& /dev/null
if [ $? != 0 ]; then
  echo "Cluster does not have proxy called 'cluster'. Nothing to do."
  exit 0
fi

#
# Remove values altogther
#
PATCH=$(mktemp /tmp/proxy-patch.XXXXXXXX)
cat >${PATCH} <<EOF
[
{
  "op": "replace",
  "path": "/spec",
  "value": {}
},
{
  "op": "replace",
  "path": "/status",
  "value": {}
}
]
EOF

kubectl patch --type='json' Proxy cluster --patch-file "${PATCH}"

if [ $? != 0 ]; then
  echo "Error: Failed to reset the Proxy"
  exit 1
fi

rm -f "${PATCH}"

set -e

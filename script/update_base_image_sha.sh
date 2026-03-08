#!/bin/bash

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

set -e

IMAGE="${1:?image argument required}"

echo "Resolving digest for ${IMAGE}..."
DIGEST=$(docker buildx imagetools inspect "$IMAGE" --format '{{json .Manifest}}' | jq -r .digest)

if [ -z "$DIGEST" ] || [ "$DIGEST" = "null" ]; then
  echo "ERROR: failed to resolve digest for ${IMAGE}"
  exit 1
fi

echo "Resolved digest: ${DIGEST}"

MAKEFILE=$(dirname "$0")/Makefile
sed -i "s|^BASE_IMAGE_SHA :=.*|BASE_IMAGE_SHA := ${DIGEST}|" "$MAKEFILE"

echo "Updated BASE_IMAGE_SHA in ${MAKEFILE}"

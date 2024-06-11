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

####
#
# This script takes care of Yaks setup
# https://github.com/citrusframework/yaks
#
####

YAKS_VERSION=0.19.2
YAKS_IMAGE=docker.io/citrusframework/yaks

set -e

# Get the os/arch
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

echo -n "Checking yaks ... "
if which yaks > /dev/null 2>&1; then
  echo "ok"
else
  echo "not found."
  YAKS_BUNDLE="linux-64bit"
  if [[ $OS == "darwin" && $ARCH == "arm64" ]]; then
    YAKS_BUNDLE="mac-arm64bit"
  fi
  YAKS_DOWNLOAD_URL="https://github.com/citrusframework/yaks/releases/download/v${YAKS_VERSION}/yaks-${YAKS_VERSION}-${YAKS_BUNDLE}.tar.gz"
  echo "Downloading ${YAKS_DOWNLOAD_URL}" && curl -f -L ${YAKS_DOWNLOAD_URL} -o yaks.tar.gz
  echo "Extracting yaks.tar.gz" && tar -zxf yaks.tar.gz

  # Install the binary using the install command
  TARGET_DIR="/usr/local/bin"
  echo "Installing yaks-${YAKS_VERSION} to ${TARGET_DIR}"
  sudo install -m 0755 yaks ${TARGET_DIR}
fi

echo "Installing the Yaks operator-image ${YAKS_IMAGE}:${YAKS_VERSION}"
yaks install --operator-image ${YAKS_IMAGE}:${YAKS_VERSION}

echo "Waiting for Yaks readiness ..."
kubectl wait --for=condition=available --timeout=300s deployment/yaks-operator
kubectl wait --for condition=Ready --timeout=300s pod -l app=yaks

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

location=$(dirname $0)
rootdir=$(realpath ${location}/..)
configdir="config"
installdir="install"
cmdutildir="cmd/util" # make cmd a hidden directory

check_env_var() {
  if [ -z "${2}" ]; then
    echo "Error: ${1} env var not defined"
    exit 1
  fi
}

check_env_var "RELEASE_VERSION" ${RELEASE_VERSION}
check_env_var "RELEASE_NAME" ${RELEASE_NAME}

pushd ${rootdir}

releasedir="${RELEASE_NAME}-${RELEASE_VERSION}-installer"
zipname="${RELEASE_NAME}-${RELEASE_VERSION}-installer.zip"

if [ -d "${releasedir}" ]; then
  rm -rf "${releasedir}"
fi
mkdir -p ${releasedir}

#
# Copies contents of install and converts softlinks
# to target files.
#
cp -rfL "${installdir}"/* "${releasedir}/"

#
# Copy the platform-check go source since its built and run during install
#
mkdir -p "${releasedir}/.${cmdutildir}/"
cp -rf "${cmdutildir}/platform-check" "${releasedir}/.${cmdutildir}/"

#
# Update location of cmd to point to hidden directory version
#
sed -i 's~^cmdutil=.*~cmdutil=\"./.cmd/util\"~' ${releasedir}/script/check_platform.sh

#
# Copy the config directory
#
cp -rfL "${configdir}" "${releasedir}/"

#
# Zip up the release
#
if [ -f "${zipname}" ]; then
  rm -f "${zipname}"
fi
zip -r "${zipname}" "${releasedir}" && rm -rf "${releasedir}"

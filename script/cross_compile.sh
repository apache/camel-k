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

location=$(dirname $0)
builddir=$(realpath ${location}/../xtmp)

rm -rf ${builddir}

basename=camel-k-client

if [ "$#" -ne 2 ]; then
    echo "usage: $0 version build_flags"
    exit 1
fi

version=$1
build_flags=$2

cross_compile () {
	local label=$1
	local extension=""
	export GOOS=$2
	export GOARCH=$3
	export CGO_ENABLED=0

	if [ "${GOOS}" == "windows" ]; then
		extension=".exe"
	fi

	targetdir=${builddir}/${label}
	eval go build "$build_flags" -o ${targetdir}/kamel${extension} ./cmd/kamel/...

	if [ -n "$GPG_PASS" ]; then
	    gpg --output ${targetdir}/kamel${extension}.asc --armor --detach-sig --passphrase ${GPG_PASS} ${targetdir}/kamel${extension}
	fi

    pushd . && cd ${targetdir} && sha512sum -b kamel${extension} > kamel${extension}.sha512 && popd

	cp ${location}/../LICENSE ${targetdir}/
	cp ${location}/../NOTICE ${targetdir}/

	pushd . && cd ${targetdir} && tar -zcvf ../../${label}.tar.gz . && popd
}

cross_compile ${basename}-${version}-linux-64bit linux amd64
cross_compile ${basename}-${version}-mac-64bit darwin amd64
cross_compile ${basename}-${version}-windows-64bit windows amd64


rm -rf ${builddir}

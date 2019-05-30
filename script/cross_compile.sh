#!/bin/bash

location=$(dirname $0)
builddir=$(realpath ${location}/../xtmp)

rm -rf ${builddir}

basename=camel-k-client

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "usage: $0 version [pgp_pass]"
    exit 1
fi

version=$1
gpg_pass=$2

cross_compile () {
	local label=$1
	local extension=""
	export GOOS=$2
	export GOARCH=$3

	if [ "${GOOS}" == "windows" ]; then
		extension=".exe"
	fi

	targetdir=${builddir}/${label}
	go build -o ${targetdir}/kamel${extension} ./cmd/kamel/...

	if [ -n "$gpg_pass" ]; then
	    gpg --output ${targetdir}/kamel${extension}.asc --armor --detach-sig --passphrase ${gpg_pass} ${targetdir}/kamel${extension}
	fi

	cp ${location}/../LICENSE ${targetdir}/
	cp ${location}/../NOTICE ${targetdir}/

	pushd . && cd ${targetdir} && tar -zcvf ../../${label}.tar.gz . && popd
}

cross_compile ${basename}-${version}-linux-64bit linux amd64
cross_compile ${basename}-${version}-mac-64bit darwin amd64
cross_compile ${basename}-${version}-windows-64bit windows amd64


rm -rf ${builddir}

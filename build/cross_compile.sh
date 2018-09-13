#!/bin/sh

location=$(dirname $0)
builddir=$(realpath $location/../xtmp)

rm -rf $builddir

basename=camel-k-client
version=$($location/get_version.sh)

cross_compile () {
	label=$1
	export GOOS=$2
	export GOARCH=$3

	targetdir=$builddir/$label
	go build -o $targetdir/kamel ./cmd/kamel/...

	cp $location/../LICENSE $targetdir/
	cp $location/../NOTICE $targetdir/

	pushd . && cd $targetdir && tar -zcvf ../../$label.tar.gz . && popd
}

cross_compile $basename-$version-linux-64bit linux amd64
cross_compile $basename-$version-mac-64bit darwin amd64
cross_compile $basename-$version-windows-64bit windows amd64


rm -rf $builddir

#!/bin/sh

location=$(dirname $0)
cat $location/../version/version.go | grep "Version" | grep "=" | awk '{print $NF}' | tr -d '"'
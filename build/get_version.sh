#!/bin/sh

location=$(dirname $0)
cat $location/../version/version.go | grep "Version" | awk '{print $NF}' | tr -d '"'
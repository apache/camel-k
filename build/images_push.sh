#!/bin/sh

location=$(dirname $0)
version=$($location/get_version.sh)

docker push docker.io/apache/camel-k:$version
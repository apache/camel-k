#!/bin/sh

location=$(dirname $0)
version=$($location/get_version.sh)

docker push ${imgDestination:-'docker.io/apache/camel-k'}:$version
#!/bin/sh

location=$(dirname $0)

if [ "$#" -ne 2 ]; then
    echo "usage: $0 version"
    exit 1
fi


docker push ${imgDestination:-'docker.io/apache/camel-k'}:$1
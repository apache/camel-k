#!/bin/sh

location=$(dirname $0)
version=$($location/get_version.sh)
IMAGE_NAME="camel-k"

if [ -z "${DOCKER_REGISTRY}" ]; then
	DOCKER_REGISTRY=$(oc get svc docker-registry -n default | awk '{b=$3 ":" substr($5, 1, match($5, "/") - 1); print b}' | tail -n+2)
fi

if [ -z "${PROJECT_NAME}" ]; then
	PROJECT_NAME=$(oc project -q)
fi

docker tag docker.io/apache/${IMAGE_NAME}:${version} ${DOCKER_REGISTRY}/${PROJECT_NAME}/${IMAGE_NAME}:${version}

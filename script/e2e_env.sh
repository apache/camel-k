#!/bin/bash

# Setup a test environment for running end-to-end (e2e) tests
#
# Create sub-shell
#
#   ./script/e2e_env.sh shell
#   $ go test -v -tags=integration ./e2e/common/languages/java_test.go
#
# Run a command in this environment
#
#   ./script/e2e_env.sh run go test -v -tags=integration ./e2e/common/languages/java_test.go
#

CAMEL_K_HOME=$(realpath "$(dirname $0)/..")
CAMEL_K_VERSION=$(grep -oE '\s+Version = ".*"' ${CAMEL_K_HOME}/pkg/util/defaults/defaults.go | cut -d '"' -f 2)
BASE_IMAGE=$(grep -oE '\s+baseImage = ".*"' ${CAMEL_K_HOME}/pkg/util/defaults/defaults.go | cut -d '"' -f 2)

# Get the current platform architecture
#
if [ "$(uname -m)" == "aarch64" ] || [ "$(uname -m)" == "arm64" ]; then
    export IMAGE_ARCH="arm64"
else
    export IMAGE_ARCH="amd64"
fi

# Get the base image sha256 from the manifest
#
BASE_IMAGE_SHA256=$(docker manifest inspect ${BASE_IMAGE} | jq -r ".manifests[] | select(.platform.architecture == \"${IMAGE_ARCH}\") | .digest")

# Export and echo the envars added by this script
#
export CAMEL_K_TEST_BASE_IMAGE="${BASE_IMAGE}@${BASE_IMAGE_SHA256}"
export CAMEL_K_TEST_OPERATOR_IMAGE="apache/camel-k:${CAMEL_K_VERSION}-${IMAGE_ARCH}"

echo "CAMEL_K_TEST_BASE_IMAGE=${CAMEL_K_TEST_BASE_IMAGE}"
echo "CAMEL_K_TEST_OPERATOR_IMAGE=${CAMEL_K_TEST_OPERATOR_IMAGE}"

# Run a shell with in this env
#
if [[ $1 == "shell" ]]; then
  export PS1='[\W]$ '
  export BASH_SILENCE_DEPRECATION_WARNING=1
  /bin/bash

# Run a command with in this env
#
elif [[ $1 == "run" ]]; then
  shift
  exec $@
fi

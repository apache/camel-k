#!/bin/bash
set +e

check_env_var() {
  if [ -z "${2}" ]; then
    echo "Error: ${1} env var not defined"
    exit 1
  fi
}

check_env_var "OPM_VERSION" ${OPM_VERSION}

OPM_PKG=github.com/operator-framework/operator-registry
#
# Timestamp for the building of the operator
#
BUILD_TIME=$(date +%Y-%m-%dT%H:%M:%S%z)

OPM_GEN_TMP_DIR=$(mktemp -d)
echo "Using temporary directory ${OPM_GEN_TMP_DIR}"
cd ${OPM_GEN_TMP_DIR}

go mod init tmp ;\
go get \
  -ldflags '-w -extldflags "-static"' -tags "json1" \
  -ldflags "-X '${OPM_PKG}/cmd/opm/version.opmVersion=${OPM_VERSION}'" \
  ${OPM_PKG}/cmd/opm@${OPM_VERSION} ;\

if [ $? != 0 ]; then
  echo "Error: Failed to install opm version ${OPM_VERSION}"
  exit 1
fi

echo "OPM Version ..."
opm version

rm -rf ${OPM_GEN_TMP_DIR}

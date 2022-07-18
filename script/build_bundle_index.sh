#!/bin/bash

check_env_var() {
  if [ -z "${2}" ]; then
    echo "Error: ${1} env var not defined"
    exit 1
  fi
}

check_env_var "BUNDLE_INDEX" ${BUNDLE_INDEX}
check_env_var "INDEX_DIR" ${INDEX_DIR}
check_env_var "PACKAGE" ${PACKAGE}
check_env_var "OPM" ${OPM}
check_env_var "BUNDLE_IMAGE" ${BUNDLE_IMAGE}
check_env_var "CSV_NAME" ${CSV_NAME}
check_env_var "CSV_REPLACES" ${CSV_REPLACES}
check_env_var "CHANNEL" ${CHANNEL}

PACKAGE_YAML=${INDEX_DIR}/${PACKAGE}.yaml

if ! command -v ${OPM} &> /dev/null
then
  echo "Error: opm is not available. Was OPM env var defined correctly: ${OPM}"
  exit 1
fi

if [ -n "${CSV_REPLACES}" ] && [ -n "${CSV_SKIPS}" ]; then
  echo
  echo "Both CSV_REPLACES and CSV_SKIPS have been specified."
  while [ -z "${brs}" ]; do
    read -p "Do you wish to include both (b), ignore 'replaces' (r) or ignore 'skips' (s): " brs
    case ${brs} in
        [Bb]* )
          echo "... including both"
          echo
          ;;
        [Rr]* )
          echo ".. ignoring 'replaces'"
          echo
          CSV_REPLACES=""
          ;;
        [Ss]* )
          echo ".. ignoring 'skips'"
          echo
          CSV_SKIPS=""
          ;;
        * )
          echo "Please answer b, r or s."
          echo
          ;;
    esac
  done
fi

if [ -f "${INDEX_DIR}.Dockerfile" ]; then
  rm -f "${INDEX_DIR}.Dockerfile"
fi

mkdir -p "${INDEX_DIR}"

if [ ! -f ${INDEX_DIR}/bundles.yaml ]; then
  ${OPM} render ${BUNDLE_INDEX} -o yaml > ${INDEX_DIR}/bundles.yaml
  if [ $? != 0 ]; then
    echo "Error: failed to render the base catalog"
    exit 1
  fi
fi

${OPM} render --skip-tls -o yaml \
  ${BUNDLE_IMAGE} > ${PACKAGE_YAML}
if [ $? != 0 ]; then
  echo "Error: failed to render the ${PACKAGE} bundle catalog"
  exit 1
fi



cat << EOF >> ${PACKAGE_YAML}
---
schema: olm.channel
package: ${PACKAGE}
name: ${CHANNEL}
entries:
  - name: ${CSV_NAME}
EOF

if [ -n "${CSV_REPLACES}" ]; then
cat << EOF >> ${PACKAGE_YAML}
    replaces: ${CSV_REPLACES}
EOF
fi

if [ -n "${CSV_SKIPS}" ]; then
cat << EOF >> ${PACKAGE_YAML}
    skipRange: "\'${CSV_SKIPS}\'"
EOF
fi

echo -n "Validating index ... "
STATUS=$(${OPM} validate ${INDEX_DIR} 2>&1)
if [ $? != 0 ]; then
  echo "Failed"
  echo "Error: ${STATUS}"
  exit 1
else
  echo "OK"
fi

echo -n "Generating catalog dockerfile ... "
STATUS=$(${OPM} generate dockerfile ${INDEX_DIR} 2>&1)
if [ $? != 0 ]; then
  echo "Failed"
  echo "Error: ${STATUS}"
  exit 1
else
  echo "OK"
fi

echo -n "Building catalog image ... "
BUNDLE_INDEX_IMAGE="${BUNDLE_IMAGE%:*}-index":"${BUNDLE_IMAGE#*:}"
STATUS=$(docker build . -f ${INDEX_DIR}.Dockerfile -t ${BUNDLE_INDEX_IMAGE} 2>&1)
if [ $? != 0 ]; then
  echo "Failed"
  echo "Error: ${STATUS}"
  exit 1
else
  echo "OK"
  echo "Index image ${BUNDLE_INDEX_IMAGE} can be pushed"
fi

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
check_env_var "YQ" ${YQ}
check_env_var "BUNDLE_IMAGE" ${BUNDLE_IMAGE}
check_env_var "CSV_NAME" ${CSV_NAME}
check_env_var "CSV_REPLACES" ${CSV_REPLACES}
check_env_var "CHANNELS" ${CHANNELS}

PACKAGE_YAML=${INDEX_DIR}/${PACKAGE}.yaml
INDEX_BASE_YAML=${INDEX_DIR}/bundles.yaml
CHANNELS_YAML="${INDEX_DIR}/${PACKAGE}-channels.yaml"

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

if [ ! -f ${INDEX_BASE_YAML} ]; then
  ${OPM} render ${BUNDLE_INDEX} -o yaml > ${INDEX_BASE_YAML}
  if [ $? != 0 ]; then
    echo "Error: failed to render the base catalog"
    exit 1
  fi
fi

if [ ! -f ${PACKAGE_YAML} ]; then
  ${OPM} render --skip-tls -o yaml \
    ${BUNDLE_IMAGE} > ${PACKAGE_YAML}
  if [ $? != 0 ]; then
    echo "Error: failed to render the ${PACKAGE} bundle catalog"
    exit 1
  fi
fi

#
# Extract the camel-k channels
#
${YQ} eval ". | select(.package == \"${PACKAGE}\" and .schema == \"olm.channel\")" ${INDEX_BASE_YAML} > ${CHANNELS_YAML}
if [ $? != 0 ] || [ ! -f "${CHANNELS_YAML}" ]; then
  echo "ERROR: Failed to extract camel-k entries from bundle catalog"
  exit 1
fi

#
# Filter out the channels in the bundles file
#
${YQ} -i eval ". | select(.package != \"${PACKAGE}\" or .schema != \"olm.channel\")" ${INDEX_BASE_YAML}
if [ $? != 0 ]; then
  echo "ERROR: Failed to remove camel-k channel entries from bundles catalog"
  exit 1
fi

#
# Split the channels and append/insert the bundle into each one
#
IFS=','
#Read the split words into an array based on comma delimiter
read -r -a CHANNEL_ARR <<< "${CHANNELS}"

for channel in "${CHANNEL_ARR[@]}";
do
  channel_props=$(${YQ} eval ". | select(.name == \"${channel}\")" ${CHANNELS_YAML})

  entry="{ \"name\": \"${CSV_NAME}\""
  if [ -n "${CSV_REPLACES}" ]; then
    entry="${entry}, \"replaces\": \"${CSV_REPLACES}\""
  fi
  if [ -n "${CSV_SKIPS}" ]; then
    entry="${entry}, \"skipRange\": \"${CSV_SKIPS}\""
  fi
  entry="${entry} }"

  if [ -z "${channel_props}" ]; then
    #
    # Append a new channel
    #
    echo "Appending channel ${channel} ..."
    object="{ \"entries\": [${entry}], \"name\": \"${channel}\", \"package\": \"${PACKAGE}\", \"schema\": \"olm.channel\" }"

    channel_file=$(mktemp ${channel}-channel-XXX.yaml)
    trap 'rm -f ${channel_file}' EXIT
    ${YQ} -n eval "${object}" > ${channel_file}

    echo "---" >> ${CHANNELS_YAML}
    cat ${channel_file} >> ${CHANNELS_YAML}
  else
    #
    # Channel already exists so insert entry
    #
    echo "Inserting channel ${channel} ..."
    ${YQ} -i eval "(. | select(.name == \"${channel}\") | .entries) += ${entry}" ${CHANNELS_YAML}
  fi
done

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

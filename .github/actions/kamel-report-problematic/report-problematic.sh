#!/bin/bash

# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

####
#
# Find all test that are labelled as problematic
#
####

set -e

while getopts ":t:" opt; do
  case "${opt}" in
    t)
      TEST_SUITE=${OPTARG}
      ;;
    :)
      echo "ERROR: Option -$OPTARG requires an argument"
      exit 1
      ;;
    \?)
      echo "ERROR: Invalid option -$OPTARG"
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

if [ -z "${TEST_SUITE}" ]; then
  echo "Error: ${0} -t <test-suite>"
  exit 1
fi

TEST_DIR="./e2e/${TEST_SUITE}"

if [ ! -d "${TEST_DIR}" ]; then
  echo "No e2e directory available ... exiting"
  exit 0
fi

PROBLEMATIC=()
while IFS= read -r -d '' f
do

  func=""
  while IFS= read -r line
  do
    if [[ "${line}" =~ ^" * " ]]; then
      continue
    elif [[ "${line}" =~ ^func* ]]; then
      func=$(echo "${line}" | sed -n "s/func \([a-zA-Z0-9]\+\).*/\1/p")
    elif [[ "${line}" =~ CAMEL_K_TEST_SKIP_PROBLEMATIC ]]; then
      PROBLEMATIC[${#PROBLEMATIC[*]}]="${f}::${func}"
    fi
  done < ${f}

done < <(find "${TEST_DIR}" -name "*.go" -print0)

if [ ${#PROBLEMATIC[*]} -gt 0 ]; then
  echo "=== Problematic Tests (${#PROBLEMATIC[*]}) ==="
  for prob in "${PROBLEMATIC[@]}"
  do
    echo "  ${prob}"
  done
  echo "==="
else
  echo "=== No Tests marked as Problematic ==="
fi

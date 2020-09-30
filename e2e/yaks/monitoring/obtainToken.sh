#!/bin/bash

SOURCE_DIR=$( dirname "${BASH_SOURCE[0]}")
TEST_FILE="${SOURCE_DIR}/metrics.feature"

TOKEN=`kubectl config view --minify --output 'jsonpath={..token}'`
sed -i -e "s/TOKEN/${TOKEN}/g" "${TEST_FILE}"
#!/bin/bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SOURCE_DIR=$( dirname "${BASH_SOURCE[0]}")
APP_FOLDER="${SOURCE_DIR}/app"

VERSION_CAMEL_K_RUNTIME=$(oc -n ${YAKS_NAMESPACE} get IntegrationPlatform camel-k -o 'jsonpath={.status.build.runtimeVersion}')
VERSION_CAMEL_QUARKUS=$(oc -n ${YAKS_NAMESPACE} get CamelCatalog camel-catalog-${VERSION_CAMEL_K_RUNTIME} -o 'jsonpath={.spec.runtime.metadata.camel-quarkus\.version}')

mvn clean install -f $APP_FOLDER -Dversion.camel.quarkus=${VERSION_CAMEL_QUARKUS}
LOCAL_MVN_HOME=$(mvn help:evaluate -Dexpression=settings.localRepository -q -DforceStdout)

OPERATOR_POD=$(oc -n ${YAKS_NAMESPACE} get pods -l name=camel-k-operator --no-headers -o custom-columns=NAME:.metadata.name)
oc -n ${YAKS_NAMESPACE} exec $OPERATOR_POD -- mkdir -p /tmp/artifacts/m2/com/github/openshift-integration/camel-k-example-metrics/1.0.0-SNAPSHOT/
oc -n ${YAKS_NAMESPACE} rsync $LOCAL_MVN_HOME/com/github/openshift-integration/camel-k-example-metrics/1.0.0-SNAPSHOT/ $OPERATOR_POD:/tmp/artifacts/m2/com/github/openshift-integration/camel-k-example-metrics/1.0.0-SNAPSHOT/ --no-perms=true
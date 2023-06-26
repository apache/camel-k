#!/bin/bash

# ---------------------------------------------------------------------------
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
# ---------------------------------------------------------------------------


set +e

resourcetypes="integrations integrationkits integrationplatforms camelcatalogs kamelets builds pipes kameletbindings"

#
# Loop through the resource types
# Cannot loop through namespace as some maybe have already been deleted so not 'visible'
#
for resourcetype in ${resourcetypes}
do
  echo "Cleaning ${resourcetype} ..."
  #
  # Find all the namespaces containing the resource type
  #
  namespaces=$(kubectl get ${resourcetype} --all-namespaces | grep -v NAMESPACE | awk '{print $1}' | sort | uniq)

  #
  # Loop through the namespaces
  #
  for ns in ${namespaces}
  do
    actives=$(kubectl get ns ${ns} &> /dev/null | grep Active)
    if [ $? == 0 ]; then
      # this namespace is still Active so do not remove resources
      continue
    fi

    printf "Removing ${resourcetype} from namespace ${ns} ... "
    ok=$(kubectl delete ${resourcetype} -n "${ns}" --all)
    if [ $? == 0 ]; then
      printf "OK\n"
    else
      printf "Error\n"
    fi
  done

done

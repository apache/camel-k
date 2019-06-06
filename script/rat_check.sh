#!/bin/sh

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

if [ "$RAT_HOME" = "" ]; then
  echo "RAT_HOME is not set, please download the Apache RAT 1.12 "
  echo "from https://creadur.apache.org/rat/download_rat.cgi,"
  echo "and export RAT_HOME once you install it."
  exit 1
fi

location=$(dirname $0)
rootdir=$(realpath $location/../)
echo $rootdir

java -jar $RAT_HOME/apache-rat-*.jar -d $rootdir -e Gopkg.lock Gopkg.toml *.adoc cluster-setup.adoc gke-setup.adoc languages.adoc traits.adoc go.mod go.sum .golangci.yml .pre-commit-config.yaml .gitignore .gitmodules .dockerignore *.json *generated*.go resources-data.txt routes.flow

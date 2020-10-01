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

# Prefer unsnapshotting to regenerating, because changes done to snapshot file may get lost

location=$(dirname $0)
olm_catalog=${location}/../deploy/olm-catalog


for d in $(find ${olm_catalog} -type d -name "*-SNAPSHOT*");
do
  mv ${d} ${d//-SNAPSHOT/}
done
for d in $(find ${olm_catalog} -type d -name "*-snapshot*");
do
  mv ${d} ${d//-snapshot/}
done

for f in $(find ${olm_catalog} -type f -name "*-SNAPSHOT*");
do
  mv ${f} ${f//-SNAPSHOT/}
done
for f in $(find ${olm_catalog} -type f -name "*-snapshot*");
do
  mv ${f} ${f//-snapshot/}
done

for f in $(find ${olm_catalog}/camel-k-dev -type f);
do
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sed -i 's/-SNAPSHOT//g' $f
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    sed -i '' 's/-SNAPSHOT//g' $f
  fi
done
for f in $(find ${olm_catalog}/camel-k-dev -type f);
do
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sed -i 's/-snapshot//g' $f
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    sed -i '' 's/-snapshot//g' $f
  fi
done

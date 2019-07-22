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

set -e

# Print JAVA_HOME
echo "Java home: $JAVA_HOME"

# First build the whole project
make

# set docker0 to promiscuous mode
sudo ip link set docker0 promisc on

# Download and install the oc binary
sudo mount --make-shared /
sudo service docker stop
sudo echo '{"insecure-registries": ["172.30.0.0/16"]}' | sudo tee /etc/docker/daemon.json > /dev/null
sudo service docker start
wget https://github.com/openshift/origin/releases/download/v$OPENSHIFT_VERSION/openshift-origin-client-tools-v$OPENSHIFT_VERSION-$OPENSHIFT_COMMIT-linux-64bit.tar.gz
tar xvzOf openshift-origin-client-tools-v$OPENSHIFT_VERSION-$OPENSHIFT_COMMIT-linux-64bit.tar.gz > oc.bin
sudo mv oc.bin /usr/local/bin/oc
sudo chmod 755 /usr/local/bin/oc

# Figure out this host's IP address
IP_ADDR="$(ip addr show eth0 | grep "inet\b" | awk '{print $2}' | cut -d/ -f1)"

# Start OpenShift
oc cluster up --public-hostname=$IP_ADDR

oc login -u system:admin

# Wait until we have a ready node in openshift
TIMEOUT=0
TIMEOUT_COUNT=60
until [ $TIMEOUT -eq $TIMEOUT_COUNT ]; do
  if [ -n "$(oc get nodes | grep Ready)" ]; then
    break
  fi

  echo "openshift is not up yet"
  let TIMEOUT=TIMEOUT+1
  sleep 5
done

if [ $TIMEOUT -eq $TIMEOUT_COUNT ]; then
  echo "Failed to start openshift"
  exit 1
fi

echo "openshift is deployed and reachable"
oc describe nodes

echo "Adding maven artifacts to the image context"
make PACKAGE_ARTIFACTS_STRATEGY=download package-artifacts

echo "Copying binary file to docker dir"
mkdir -p ./build/_output/bin
cp ./camel-k ./build/_output/bin/
cp ./builder ./build/_output/bin/

echo "Building the images"
export IMAGE=docker.io/apache/camel-k:$(make version)
docker build -t "${IMAGE}" -f build/Dockerfile .

echo "installing camel k cluster resources"
./kamel install --cluster-setup

oc login -u developer

# Then run integration tests
make test-integration


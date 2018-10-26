#!/bin/sh

set -e

# Find the JAVA_HOME and set the KOTLIN_JDK_HOME
echo "Java home: $JAVA_HOME"
export KOTLIN_JDK_HOME=$JAVA_HOME

# First build the whole project
make

# set docker0 to promiscuous mode
sudo ip link set docker0 promisc on

# Download and install the oc binary
sudo mount --make-shared /
sudo service docker stop
sudo sed -i 's/DOCKER_OPTS=\"/DOCKER_OPTS=\"--insecure-registry 172.30.0.0\/16 /' /etc/default/docker
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
./build/package_maven_artifacts.sh

echo "Copying binary file to docker dir"
mkdir -p ./tmp/_output/bin
cp ./camel-k-operator ./tmp/_output/bin/

echo "Building the images"
export IMAGE=docker.io/apache/camel-k:$(./build/get_version.sh)
./tmp/build/docker_build.sh

echo "installing camel k cluster resources"
./kamel install --cluster-setup

oc login -u developer


# Then run integration tests
make test-integration

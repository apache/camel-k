#!/bin/sh

# Exit on error
set -e

if [ "$project" = "" ]; then
  project=$(oc project -q)
else
  oc new-project $project 2>/dev/null || true
fi

# Compile and build images
make
eval $(minishift docker-env)
make images

# Try setup with standard user
ret=0
./kamel install -n $project 2>/dev/null || export ret=$?

if [ $ret -ne 0 ]; then
  # Login as admin if cluster setup fails with standard user
  olduser=$(oc whoami)
  oc login -u system:admin
  ./kamel install --cluster-setup
  oc login -u $olduser
  ./kamel install -n $project
fi

oc delete pod -l name=camel-k-operator -n $project || true

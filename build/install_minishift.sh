#!/bin/sh

oc login -u system:admin
make
eval $(minishift docker-env)
make images
./kamel install --cluster-setup
oc delete pod -l name=camel-k-operator
oc login -u developer
./kamel install


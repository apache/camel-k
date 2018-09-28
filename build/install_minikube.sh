#!/bin/sh

# Exit on error
set -e

# Compile and build images
make
eval $(minikube docker-env)
make images

# Perform installation
./kamel install

kubectl delete pod -l name=camel-k-operator || true

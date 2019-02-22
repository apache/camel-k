#!/bin/sh

# Exit on error
set -e

eval $(minikube docker-env)
make images

# Perform installation
./kamel install

kubectl delete pod -l name=camel-k-operator || true

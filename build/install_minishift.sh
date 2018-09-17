#!/bin/sh

if [ "$project" = "" ]; then
  project="myproject"
fi
echo $project
if [ "$project" != "myproject" ]; then
  oc new-project $project
  oc project $project
fi
oc login -u system:admin 
make
eval $(minishift docker-env)
make images
./kamel install --cluster-setup
oc delete pod -l name=camel-k-operator
oc login -u developer 
if [ "$project" != "myproject" ]; then
  oc project $project
fi
./kamel install


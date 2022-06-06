#!/bin/bash

#
# Remove any operator groups that might be left over from
# previous failed tests and conflict with operator installation
#

set +e

#
# Find all the namespaces containing the operator groups
# that may exist in other test-* namespaces
#
namespaces=$(kubectl get operatorgroups --all-namespaces | grep -v NAMESPACE | awk '{print $1}' | grep "test-")
if [ -z "${namespaces}" ]; then
  echo "No test operatorgroups installed"
  exit 0
fi

echo "Test namespaces with operatorgroups: ${namespaces}"

#
# Loop through the namespaces
#
for ns in ${namespaces}
do
  echo "Examining namespace ${ns}"

  #
  # Find all the resources of the type in the namespace
  #
  ogs=$(kubectl get operatorgroups -n "${ns}" | grep -v NAME | awk '{print $1}')
  echo "Identified operatorgroups:  ${ogs}"

  #
  # Loop through the groups
  #
  for og in ${ogs}
  do
    echo "Removing operatorgroup / ${og} from namespace ${ns} ... "
    kubectl delete operatorgroup ${og} -n "${ns}"
  done

done

ogs=$(kubectl get operatorgroups --all-namespaces)
if [ -z "${ogs}" ]; then
  echo "No operatorgroups remaining"
else
  echo "Remaining namespaces with operatorgroups: \"${namespaces}\""
fi

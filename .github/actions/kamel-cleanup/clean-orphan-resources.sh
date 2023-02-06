#!/bin/bash

set +e

resourcetypes="integrations integrationkits integrationplatforms camelcatalogs kamelets builds kameletbindings"

#
# Loop through the resource types
# Cannot loop through namespace as some maybe have already been deleted so not 'visible'
#
for resourcetype in ${resourcetypes}
do
  echo "Cleaning ${resourcetype} ..."
  #
  # Find all the namespaces containing the resource type
  #
  namespaces=$(kubectl get ${resourcetype} --all-namespaces | grep -v NAMESPACE | awk '{print $1}' | sort | uniq)

  #
  # Loop through the namespaces
  #
  for ns in ${namespaces}
  do
    actives=$(kubectl get ns ${ns} &> /dev/null | grep Active)
    if [ $? == 0 ]; then
      # this namespace is still Active so do not remove resources
      continue
    fi

    printf "Removing ${resourcetype} from namespace ${ns} ... "
    ok=$(kubectl delete ${resourcetype} -n "${ns}" --all)
    if [ $? == 0 ]; then
      printf "OK\n"
    else
      printf "Error\n"
    fi
  done

done

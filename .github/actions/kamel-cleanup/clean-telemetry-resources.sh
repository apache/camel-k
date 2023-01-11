#!/bin/bash

#
# Remove any telemetry groups resources that might have been deployed for tests.
# All the telemetry resources are deployed in otlp namespace.
#

set +e

#
# Find if the namespace containing telemetry resources exists
#
namespace_otlp=$(kubectl --ignore-not-found=true get namespaces otlp)
if [ -z "${namespace_otlp}" ]; then
  echo "No telemetry resource installed"
  exit 0
fi

echo "Telemetry namespace exists: ${namespace_otlp}"

#
# Delete telemetry resources namespace
#
kubectl delete --now --timeout=600s namespace ${namespace_otlp} 1> /dev/null

echo "Telemetry resources deleted"
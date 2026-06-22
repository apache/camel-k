#!/bin/bash

# ---------------------------------------------------------------------------
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ---------------------------------------------------------------------------

# Installs the namespaced Camel K operator RBAC (Role + RoleBinding) into a namespace that an
# operator running in a *different* namespace should watch. This is the per-namespace step of the
# multi-namespace / dynamic-namespace installation: it grants a remote operator the rights it needs
# in this namespace, while keeping its access scoped to exactly the namespaces you opt in.
#
# The RoleBinding subjects are set to the operator ServiceAccount(s) in the operator namespace, so the
# Role is granted to the remote operator(s) (not to a ServiceAccount in the watched namespace).
#
# Multiple operator ServiceAccounts can be passed as a comma-separated list. This is the "all shards
# watch every namespace, reconcile by operator.id" model: every operator shard shares the same
# namespaced Role in a watched namespace via a single multi-subject RoleBinding. (Work is partitioned
# at runtime by the camel.apache.org/operator.id annotation, not by RBAC.)
#
# Usage:
#   ./script/install_namespace_rbac.sh <watched-namespace> [operator-namespace] [operator-sa[,operator-sa...]]
#
#   watched-namespace   (required) namespace to install the operator RBAC into.
#   operator-namespace  (optional) namespace where the operator(s) run. Default: camel-k.
#   operator-sa         (optional) operator ServiceAccount name, or a comma-separated list of them
#                       (one per shard). Default: camel-k-operator.
#
# Examples:
#   # single operator
#   ./script/install_namespace_rbac.sh team-a camel-k camel-k-operator
#   # two shards both able to watch team-a
#   ./script/install_namespace_rbac.sh team-a camel-k camel-k-shard-1-operator,camel-k-shard-2-operator
#
# Requires: kubectl and kustomize on the PATH, and rights to create Roles/RoleBindings in the
# target namespace.

set -e

location=$(dirname "$0")
rootdir=$(realpath "${location}/../")

watched_namespace="$1"
operator_namespace="${2:-camel-k}"
operator_sas="${3:-camel-k-operator}"

if [ -z "${watched_namespace}" ]; then
  echo "Error: watched namespace is required."
  echo "Usage: $0 <watched-namespace> [operator-namespace] [operator-sa[,operator-sa...]]"
  exit 1
fi

# Build the RoleBinding subjects list: one ServiceAccount entry per (comma-separated) operator shard,
# each pinned to the operator namespace. Indented to sit under the JSON6902 patch "value:" key below.
subjects_block=""
sa_summary=""
IFS=',' read -ra _sa_arr <<< "${operator_sas}"
for _sa in "${_sa_arr[@]}"; do
  _sa="$(echo -n "${_sa}" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
  [ -z "${_sa}" ] && continue
  subjects_block="${subjects_block}      - kind: ServiceAccount
        name: ${_sa}
        namespace: ${operator_namespace}
"
  sa_summary="${sa_summary}${sa_summary:+, }${_sa}"
done

if [ -z "${subjects_block}" ]; then
  echo "Error: at least one operator ServiceAccount is required."
  exit 1
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "${tmpdir}"' EXIT

# Kustomize requires relative resource paths, so copy the canonical namespaced RBAC into the temp
# directory and reference it relatively. This keeps a single source of truth for the role rules.
mkdir -p "${tmpdir}/base"
cp "${rootdir}/pkg/resources/config/rbac/namespaced/"*.yaml "${tmpdir}/base/"

cat > "${tmpdir}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: ${watched_namespace}
resources:
- ./base
patches:
# The operator ServiceAccount(s) live in the operator namespace, not in the watched namespace. Replace
# the whole subjects list so every operator shard is granted the namespaced Role via one RoleBinding.
- target:
    kind: RoleBinding
  patch: |-
    - op: replace
      path: /subjects
      value:
${subjects_block}
EOF

echo "Installing Camel K operator RBAC into namespace '${watched_namespace}'"
echo "  bound to ServiceAccount(s) [${sa_summary}] in namespace '${operator_namespace}'"
kustomize build "${tmpdir}" | kubectl apply -f -

echo "Done. The operator in '${operator_namespace}' can now watch namespace '${watched_namespace}'."
echo "If you use dynamic discovery, label the namespace to start watching it:"
echo "  kubectl label namespace ${watched_namespace} camel-k-enabled=true"

#!/usr/bin/env bash
set -euo pipefail

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

namespace=keda
service_dns=azure-servicebus-emulator.keda.svc.cluster.local
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# KEDA 2.17.2's Go Azure SDK uses HTTPS for the Service Bus administration API.
# The Service Bus emulator exposes that API over plain HTTP. This test-only
# proxy supplies HTTPS with a short-lived certificate and forwards requests to
# the emulator's HTTP management port. Only the KEDA operator trusts this CA.
openssl req -x509 -newkey rsa:2048 -nodes -days 2 \
  -keyout "$tmpdir/ca.key" \
  -out "$tmpdir/ca.crt" \
  -subj "/CN=Camel K KEDA emulator test CA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"

openssl req -newkey rsa:2048 -nodes \
  -keyout "$tmpdir/tls.key" \
  -out "$tmpdir/tls.csr" \
  -subj "/CN=$service_dns"

cat > "$tmpdir/tls.ext" <<EOF
subjectAltName=DNS:azure-servicebus-emulator,DNS:azure-servicebus-emulator.keda,DNS:azure-servicebus-emulator.keda.svc,DNS:$service_dns
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
basicConstraints=critical,CA:FALSE
EOF

openssl x509 -req -in "$tmpdir/tls.csr" \
  -CA "$tmpdir/ca.crt" -CAkey "$tmpdir/ca.key" -CAcreateserial \
  -out "$tmpdir/tls.crt" -days 2 -sha256 -extfile "$tmpdir/tls.ext"

kubectl -n "$namespace" create secret tls azure-servicebus-emulator-tls \
  --cert="$tmpdir/tls.crt" --key="$tmpdir/tls.key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "$namespace" create configmap azure-servicebus-emulator-ca \
  --from-file=ca.crt="$tmpdir/ca.crt" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f ./e2e/keda/setup/azure-servicebus-emulator.yaml
kubectl rollout status deployment/azure-servicebus-emulator -n "$namespace" --timeout=10m

# Azure scaler construction and metric queries run inside keda-operator (not
# keda-metrics-apiserver). Keep the system trust directory and add the test CA.
kubectl patch deployment keda-operator -n "$namespace" --type=strategic --patch='
spec:
  template:
    spec:
      containers:
        - name: keda-operator
          env:
            - name: SSL_CERT_DIR
              value: /etc/ssl/certs:/var/run/azure-servicebus-ca
          volumeMounts:
            - name: azure-servicebus-emulator-ca
              mountPath: /var/run/azure-servicebus-ca
              readOnly: true
      volumes:
        - name: azure-servicebus-emulator-ca
          configMap:
            name: azure-servicebus-emulator-ca
'

kubectl rollout status deployment/keda-operator -n "$namespace" --timeout=180s

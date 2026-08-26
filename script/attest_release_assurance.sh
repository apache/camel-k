#!/usr/bin/env bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

receipt=${1:-release-assurance.json}
bundle=${2:-release-assurance.sigstore.json}
issuer=${COSIGN_CERTIFICATE_OIDC_ISSUER:-https://token.actions.githubusercontent.com}

for command in cosign jq; do
    command -v "$command" >/dev/null 2>&1 || {
        echo "required command not found: $command" >&2
        exit 1
    }
done

if [[ ! -f "$receipt" ]]; then
    echo "release assurance receipt not found: $receipt" >&2
    exit 1
fi

identity=${COSIGN_CERTIFICATE_IDENTITY:-}
if [[ -z "$identity" ]]; then
    if [[ -z "${GITHUB_WORKFLOW_REF:-}" ]]; then
        echo "GITHUB_WORKFLOW_REF or COSIGN_CERTIFICATE_IDENTITY is required for keyless verification" >&2
        exit 2
    fi
    identity="https://github.com/${GITHUB_WORKFLOW_REF}"
fi

repository_without_tag() {
    local ref=${1%@*}
    local tail=${ref##*/}

    if [[ "$tail" == *:* ]]; then
        ref=${ref%:*}
    fi

    printf '%s\n' "$ref"
}

verify_identity_args=(
    --certificate-identity "$identity"
    --certificate-oidc-issuer "$issuer"
)

export COSIGN_YES=true

# The local bundle makes the downloadable receipt independently verifiable,
# while OCI signatures and attestations make the registry artifacts discoverable.
cosign sign-blob --yes --bundle "$bundle" "$receipt"
cosign verify-blob \
    --bundle "$bundle" \
    "${verify_identity_args[@]}" \
    "$receipt" >/dev/null

while IFS= read -r image; do
    tag_ref=$(jq -r '.ref' <<<"$image")
    digest=$(jq -r '.digest' <<<"$image")
    repository=$(repository_without_tag "$tag_ref")
    subject="${repository}@${digest}"

    if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
        echo "invalid image digest in receipt: $digest" >&2
        exit 1
    fi

    cosign sign --yes "$subject"
    cosign verify "${verify_identity_args[@]}" "$subject" >/dev/null

    cosign attest --yes --type custom --predicate "$receipt" "$subject"
    cosign verify-attestation \
        --type custom \
        "${verify_identity_args[@]}" \
        "$subject" >/dev/null

    while IFS= read -r sbom; do
        sbom_name=$(jq -r '.name' <<<"$sbom")
        sbom_source=$(jq -r '.source' <<<"$sbom")
        sbom_digest=$(jq -r '.imageDigest' <<<"$sbom")

        if [[ ! -f "$sbom_name" ]]; then
            echo "image SBOM not found: $sbom_name" >&2
            exit 1
        fi
        if [[ "$sbom_source" != *@"$sbom_digest" ]]; then
            echo "image SBOM source/digest binding mismatch: $sbom_name" >&2
            exit 1
        fi

        cosign attest --yes --type cyclonedx --predicate "$sbom_name" "$sbom_source"
        cosign verify-attestation \
            --type cyclonedx \
            "${verify_identity_args[@]}" \
            "$sbom_source" >/dev/null
    done < <(jq -c '.sboms[]?' <<<"$image")
done < <(jq -c '.images[]' "$receipt")

# Re-run the entire release contract from the consumer side. Strict mode requires
# every local release asset, recomputes hashes, re-reads embedded Go VCS metadata,
# and requires the verified OCI predicates to match the exact local JSON files.
ASSURANCE_REQUIRE_LOCAL_ASSETS=true \
COSIGN_CERTIFICATE_IDENTITY="$identity" \
COSIGN_CERTIFICATE_OIDC_ISSUER="$issuer" \
bash "$(dirname "$0")/verify_release_assurance.sh" \
    "$receipt" \
    "$bundle" \
    release-assurance.verdict.json

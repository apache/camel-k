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

set -uo pipefail

mode=receipt
if [[ ${1:-} == "--image" ]]; then
  mode=image
  shift
fi

issuer=${COSIGN_CERTIFICATE_OIDC_ISSUER:-https://token.actions.githubusercontent.com}
identity=${COSIGN_CERTIFICATE_IDENTITY:-}
if [[ -z "$identity" && -n "${GITHUB_WORKFLOW_REF:-}" ]]; then
  identity="https://github.com/${GITHUB_WORKFLOW_REF}"
fi

verdict_file=${ASSURANCE_VERDICT_FILE:-release-assurance.verdict.json}
require_local=${ASSURANCE_REQUIRE_LOCAL_ASSETS:-false}
errors_file=$(mktemp)
checks_file=$(mktemp)
warnings_file=$(mktemp)
trap 'rm -f "$errors_file" "$checks_file" "$warnings_file"' EXIT
: >"$errors_file"
: >"$checks_file"
: >"$warnings_file"

record_check() {
  jq -cn --arg name "$1" --arg status "$2" --arg detail "$3" \
    '{name:$name,status:$status,detail:$detail}' >>"$checks_file"
}
reject() { printf '%s\n' "$1" >>"$errors_file"; record_check "$2" "fail" "$1"; }
pass() { record_check "$1" "pass" "$2"; }
warn() { printf '%s\n' "$1" >>"$warnings_file"; record_check "$2" "warn" "$1"; }

finish() {
  local verdict="TRUST"
  [[ -s "$errors_file" ]] && verdict="REJECT"
  jq -s '.' "$checks_file" >"${checks_file}.json"
  if [[ -s "$errors_file" ]]; then
    jq -R -s 'split("\n")[:-1]' "$errors_file" >"${errors_file}.json"
  else
    echo '[]' >"${errors_file}.json"
  fi
  if [[ -s "$warnings_file" ]]; then
    jq -R -s 'split("\n")[:-1]' "$warnings_file" >"${warnings_file}.json"
  else
    echo '[]' >"${warnings_file}.json"
  fi
  jq -n \
    --arg verdict "$verdict" \
    --arg mode "$mode" \
    --slurpfile checks "${checks_file}.json" \
    --slurpfile errors "${errors_file}.json" \
    --slurpfile warnings "${warnings_file}.json" \
    '{verdict:$verdict,mode:$mode,checks:$checks[0],errors:$errors[0],warnings:$warnings[0]}' \
    >"$verdict_file"
  cat "$verdict_file"
  [[ "$verdict" == TRUST ]]
}

for command in jq cosign base64; do
  if ! command -v "$command" >/dev/null 2>&1; then
    reject "required command not found: $command" "tool:$command"
  else
    pass "tool:$command" "available"
  fi
done
if command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1; then
  pass "tool:sha256" "available"
else
  reject "required command not found: sha256sum or shasum" "tool:sha256"
fi
if [[ -z "$identity" ]]; then
  reject "COSIGN_CERTIFICATE_IDENTITY or GITHUB_WORKFLOW_REF is required" "identity"
else
  pass "identity" "$identity"
fi
if [[ -s "$errors_file" ]]; then
  finish
  exit $?
fi

verify_identity_args=(
  --certificate-identity "$identity"
  --certificate-oidc-issuer "$issuer"
)

base64_decode() {
  if base64 --decode </dev/null >/dev/null 2>&1; then
    base64 --decode
  else
    base64 -D
  fi
}

sha256_stdin() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
  else
    shasum -a 256 | awk '{print $1}'
  fi
}

canonical_json_hash() {
  jq -cS . "$1" | sha256_stdin
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

repository_without_tag() {
  local ref=${1%@*}
  local tail=${ref##*/}
  [[ "$tail" == *:* ]] && ref=${ref%:*}
  printf '%s\n' "$ref"
}

verify_attested_json_file() {
  local type=$1
  local subject=$2
  local file=$3
  local name=$4
  local out target_hash matched=0

  if ! out=$(cosign verify-attestation --type "$type" "${verify_identity_args[@]}" "$subject" 2>/dev/null); then
    reject "attestation verification failed for $subject ($type)" "$name:signature"
    return
  fi

  target_hash=$(canonical_json_hash "$file") || {
    reject "cannot canonicalize $file" "$name:local"
    return
  }

  while IFS= read -r payload; do
    [[ -z "$payload" ]] && continue
    local data tmp hash
    data=$(printf '%s' "$payload" | base64_decode 2>/dev/null | jq -r '.predicate.Data // empty' 2>/dev/null || true)
    [[ -z "$data" ]] && continue
    tmp=$(mktemp)
    printf '%s' "$data" >"$tmp"
    if jq -e . "$tmp" >/dev/null 2>&1; then
      hash=$(canonical_json_hash "$tmp")
      [[ "$hash" == "$target_hash" ]] && matched=1
    fi
    rm -f "$tmp"
  done < <(printf '%s\n' "$out" | jq -r \
    'if type == "array" then .[]?.payload // empty else .payload // empty end' 2>/dev/null)

  if [[ $matched -eq 1 ]]; then
    pass "$name" "signed predicate matches local JSON"
  else
    reject "no verified $type attestation matches local $file" "$name"
  fi
}

if [[ "$mode" == image ]]; then
  subject=${1:-}
  verdict_file=${2:-$verdict_file}

  if [[ ! "$subject" =~ @sha256:[0-9a-f]{64}$ ]]; then
    reject "image subject must be immutable @sha256 reference" "image:subject"
    finish
    exit $?
  fi

  if cosign verify "${verify_identity_args[@]}" "$subject" >/dev/null 2>&1; then
    pass "image:signature" "$subject"
  else
    reject "image signature verification failed: $subject" "image:signature"
  fi
  if cosign verify-attestation --type custom "${verify_identity_args[@]}" "$subject" >/dev/null 2>&1; then
    pass "image:receipt-attestation" "present"
  else
    reject "release receipt attestation missing or invalid: $subject" "image:receipt-attestation"
  fi
  if cosign verify-attestation --type cyclonedx "${verify_identity_args[@]}" "$subject" >/dev/null 2>&1; then
    pass "image:sbom-attestation" "present"
  else
    reject "CycloneDX attestation missing or invalid: $subject" "image:sbom-attestation"
  fi

  finish
  exit $?
fi

receipt=${1:-release-assurance.json}
bundle=${2:-release-assurance.sigstore.json}
verdict_file=${3:-$verdict_file}
asset_dir=${ASSURANCE_ASSET_DIR:-$(dirname "$receipt")}

if [[ ! -f "$receipt" ]]; then
  reject "receipt not found: $receipt" "receipt:file"
  finish
  exit $?
fi

if jq -e '.schemaVersion == 1 and (.images|type=="array") and (.artifacts|type=="array")' \
  "$receipt" >/dev/null 2>&1; then
  pass "receipt:schema" "schemaVersion=1"
else
  reject "invalid or unsupported receipt schema" "receipt:schema"
fi

if [[ ! -f "$bundle" ]]; then
  reject "Sigstore bundle not found: $bundle" "receipt:bundle"
elif cosign verify-blob --bundle "$bundle" "${verify_identity_args[@]}" "$receipt" >/dev/null 2>&1; then
  pass "receipt:signature" "Sigstore bundle verified"
else
  reject "receipt Sigstore bundle verification failed" "receipt:signature"
fi

sidecar="${receipt}.sha256"
[[ "$receipt" == release-assurance.json ]] && \
  sidecar="$(dirname "$receipt")/release-assurance.json.sha256"
if [[ -f "$sidecar" ]]; then
  expected=$(awk '{print $1}' "$sidecar")
  actual=$(sha256_file "$receipt")
  if [[ "$expected" == "$actual" ]]; then
    pass "receipt:sha256" "$actual"
  else
    reject "receipt SHA-256 mismatch" "receipt:sha256"
  fi
elif [[ "$require_local" == true ]]; then
  reject "receipt SHA-256 sidecar missing" "receipt:sha256"
else
  warn "receipt SHA-256 sidecar not present" "receipt:sha256"
fi

build_sha=$(jq -r '.release.buildSourceSha // empty' "$receipt")
release_sha=$(jq -r '.release.releaseCommitSha // empty' "$receipt")
[[ "$build_sha" =~ ^[0-9a-f]{40,64}$ ]] && \
  pass "release:build-source" "$build_sha" || reject "invalid buildSourceSha" "release:build-source"
[[ "$release_sha" =~ ^[0-9a-f]{40,64}$ ]] && \
  pass "release:commit" "$release_sha" || reject "invalid releaseCommitSha" "release:commit"

module_name=$(jq -r '.moduleSbom.name // empty' "$receipt")
if [[ -n "$module_name" ]]; then
  file="$asset_dir/$(basename "$module_name")"
  if [[ -f "$file" ]]; then
    expected=$(jq -r '.moduleSbom.sha256' "$receipt")
    actual=$(sha256_file "$file")
    [[ "$expected" == "$actual" ]] && \
      pass "module-sbom:sha256" "$actual" || reject "module SBOM hash mismatch" "module-sbom:sha256"
  elif [[ "$require_local" == true ]]; then
    reject "module SBOM missing: $file" "module-sbom:file"
  else
    warn "module SBOM not downloaded: $file" "module-sbom:file"
  fi
fi

while IFS= read -r artifact; do
  name=$(jq -r '.name' <<<"$artifact")
  sbom=$(jq -r '.sbom.name' <<<"$artifact")
  source_sha=$(jq -r '.binarySourceSha' <<<"$artifact")

  [[ "$source_sha" == "$build_sha" ]] && \
    pass "artifact:$name:source" "$source_sha" || \
    reject "binary source SHA does not match build source for $name" "artifact:$name:source"

  expected=$(jq -r '.sha256' <<<"$artifact")
  file="$asset_dir/$(basename "$name")"
  if [[ -f "$file" ]]; then
    actual=$(sha256_file "$file")
    [[ "$actual" == "$expected" ]] && \
      pass "artifact:$name:sha256" "$actual" || reject "hash mismatch: $name" "artifact:$name:sha256"

    if command -v tar >/dev/null 2>&1 && command -v go >/dev/null 2>&1; then
      extract_dir=$(mktemp -d)
      binary_name=kamel
      target=$(jq -r '.target // empty' <<<"$artifact")
      [[ "$target" == windows-* ]] && binary_name=kamel.exe
      if tar -xzf "$file" -C "$extract_dir" "$binary_name" >/dev/null 2>&1; then
        embedded=$(go version -m "$extract_dir/$binary_name" 2>/dev/null | \
          sed -n 's/^[[:space:]]*build[[:space:]]*vcs\.revision=//p' | head -n 1)
        [[ "$embedded" == "$build_sha" ]] && \
          pass "artifact:$name:embedded-source" "$embedded" || \
          reject "embedded binary source revision mismatch: $name" "artifact:$name:embedded-source"
      else
        reject "cannot extract release binary from $name" "artifact:$name:embedded-source"
      fi
      rm -rf "$extract_dir"
    elif [[ "$require_local" == true ]]; then
      reject "go and tar are required for strict binary source verification" "artifact:$name:embedded-source"
    else
      warn "go/tar unavailable; embedded source revision not independently checked" \
        "artifact:$name:embedded-source"
    fi
  elif [[ "$require_local" == true ]]; then
    reject "local asset missing: $name" "artifact:$name:file"
  else
    warn "local asset not downloaded: $name" "artifact:$name:file"
  fi

  expected=$(jq -r '.sbom.sha256' <<<"$artifact")
  file="$asset_dir/$(basename "$sbom")"
  if [[ -f "$file" ]]; then
    actual=$(sha256_file "$file")
    [[ "$actual" == "$expected" ]] && \
      pass "artifact:$name:sbom:sha256" "$actual" || reject "hash mismatch: $sbom" "artifact:$name:sbom:sha256"
  elif [[ "$require_local" == true ]]; then
    reject "local asset missing: $sbom" "artifact:$name:sbom:file"
  else
    warn "local asset not downloaded: $sbom" "artifact:$name:sbom:file"
  fi
done < <(jq -c '.artifacts[]' "$receipt")

while IFS= read -r image; do
  kind=$(jq -r '.kind' <<<"$image")
  ref=$(jq -r '.ref' <<<"$image")
  digest=$(jq -r '.digest' <<<"$image")
  repository=$(repository_without_tag "$ref")
  subject="${repository}@${digest}"

  if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    reject "invalid image digest for $kind" "image:$kind:digest"
    continue
  fi

  if cosign verify "${verify_identity_args[@]}" "$subject" >/dev/null 2>&1; then
    pass "image:$kind:signature" "$subject"
  else
    reject "image signature verification failed: $subject" "image:$kind:signature"
  fi

  verify_attested_json_file custom "$subject" "$receipt" "image:$kind:receipt-attestation"

  while IFS= read -r sbom; do
    name=$(jq -r '.name' <<<"$sbom")
    source=$(jq -r '.source' <<<"$sbom")
    digest=$(jq -r '.imageDigest' <<<"$sbom")
    expected=$(jq -r '.sha256' <<<"$sbom")
    file="$asset_dir/$(basename "$name")"

    [[ "$source" == *@"$digest" ]] && \
      pass "image:$kind:sbom:$name:binding" "$source" || \
      reject "SBOM source/digest mismatch: $name" "image:$kind:sbom:$name:binding"

    if [[ -f "$file" ]]; then
      actual=$(sha256_file "$file")
      [[ "$actual" == "$expected" ]] && \
        pass "image:$kind:sbom:$name:sha256" "$actual" || \
        reject "image SBOM hash mismatch: $name" "image:$kind:sbom:$name:sha256"
      verify_attested_json_file cyclonedx "$source" "$file" \
        "image:$kind:sbom:$name:attestation"
    elif [[ "$require_local" == true ]]; then
      reject "image SBOM missing: $file" "image:$kind:sbom:$name:file"
    else
      warn "image SBOM not downloaded; verifying attestation existence only: $name" \
        "image:$kind:sbom:$name:file"
      if cosign verify-attestation --type cyclonedx "${verify_identity_args[@]}" \
        "$source" >/dev/null 2>&1; then
        pass "image:$kind:sbom:$name:attestation" "verified"
      else
        reject "CycloneDX attestation verification failed: $source" \
          "image:$kind:sbom:$name:attestation"
      fi
    fi
  done < <(jq -c '.sboms[]?' <<<"$image")
done < <(jq -c '.images[]' "$receipt")

finish

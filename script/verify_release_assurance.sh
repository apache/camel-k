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

receipt=${1:-release-assurance.json}
verdict_file=${2:-${ASSURANCE_VERDICT_FILE:-release-assurance.verdict.json}}
asset_dir=${ASSURANCE_ASSET_DIR:-$(dirname "$receipt")}
require_local=${ASSURANCE_REQUIRE_LOCAL_ASSETS:-false}

errors_file=$(mktemp)
checks_file=$(mktemp)
warnings_file=$(mktemp)
trap 'rm -f "$errors_file" "$checks_file" "$warnings_file" "${errors_file}.json" "${checks_file}.json" "${warnings_file}.json"' EXIT
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
  local verdict="VALID"
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
    --arg assuranceClass "unsigned-nightly-integrity" \
    --slurpfile checks "${checks_file}.json" \
    --slurpfile errors "${errors_file}.json" \
    --slurpfile warnings "${warnings_file}.json" \
    '{verdict:$verdict,assuranceClass:$assuranceClass,checks:$checks[0],errors:$errors[0],warnings:$warnings[0]}' \
    >"$verdict_file"

  cat "$verdict_file"
  [[ "$verdict" == VALID ]]
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

for command in jq; do
  if command -v "$command" >/dev/null 2>&1; then
    pass "tool:$command" "available"
  else
    reject "required command not found: $command" "tool:$command"
  fi
done

if command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1; then
  pass "tool:sha256" "available"
else
  reject "required command not found: sha256sum or shasum" "tool:sha256"
fi

if [[ ! -f "$receipt" ]]; then
  reject "receipt not found: $receipt" "receipt:file"
  finish
  exit $?
fi

if [[ -s "$errors_file" ]]; then
  finish
  exit $?
fi

if jq -e '
  .schemaVersion == 1 and
  (.release | type == "object") and
  (.images | type == "array") and
  (.artifacts | type == "array")
' "$receipt" >/dev/null 2>&1; then
  pass "receipt:schema" "schemaVersion=1"
else
  reject "invalid or unsupported receipt schema" "receipt:schema"
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
if [[ "$build_sha" =~ ^[0-9a-f]{40,64}$ ]]; then
  pass "release:build-source" "$build_sha"
else
  reject "invalid buildSourceSha" "release:build-source"
fi
if [[ "$release_sha" =~ ^[0-9a-f]{40,64}$ ]]; then
  pass "release:commit" "$release_sha"
else
  reject "invalid releaseCommitSha" "release:commit"
fi

module_name=$(jq -r '.moduleSbom.name // empty' "$receipt")
if [[ -n "$module_name" ]]; then
  module_file="$asset_dir/$(basename "$module_name")"
  if [[ -f "$module_file" ]]; then
    expected=$(jq -r '.moduleSbom.sha256 // empty' "$receipt")
    actual=$(sha256_file "$module_file")
    if [[ "$expected" == "$actual" ]]; then
      pass "module-sbom:sha256" "$actual"
    else
      reject "module SBOM hash mismatch" "module-sbom:sha256"
    fi
  elif [[ "$require_local" == true ]]; then
    reject "module SBOM missing: $module_file" "module-sbom:file"
  else
    warn "module SBOM not downloaded: $module_file" "module-sbom:file"
  fi
fi

while IFS= read -r artifact; do
  name=$(jq -r '.name' <<<"$artifact")
  sbom=$(jq -r '.sbom.name' <<<"$artifact")
  source_sha=$(jq -r '.binarySourceSha' <<<"$artifact")

  if [[ "$source_sha" == "$build_sha" ]]; then
    pass "artifact:$name:source" "$source_sha"
  else
    reject "binary source SHA does not match build source for $name" "artifact:$name:source"
  fi

  artifact_file="$asset_dir/$(basename "$name")"
  expected=$(jq -r '.sha256' <<<"$artifact")
  if [[ -f "$artifact_file" ]]; then
    actual=$(sha256_file "$artifact_file")
    if [[ "$actual" == "$expected" ]]; then
      pass "artifact:$name:sha256" "$actual"
    else
      reject "hash mismatch: $name" "artifact:$name:sha256"
    fi

    if command -v tar >/dev/null 2>&1 && command -v go >/dev/null 2>&1; then
      extract_dir=$(mktemp -d)
      binary_name=kamel
      target=$(jq -r '.target // empty' <<<"$artifact")
      [[ "$target" == windows-* ]] && binary_name=kamel.exe
      if tar -xzf "$artifact_file" -C "$extract_dir" "$binary_name" >/dev/null 2>&1; then
        embedded=$(go version -m "$extract_dir/$binary_name" 2>/dev/null | \
          sed -n 's/^[[:space:]]*build[[:space:]]*vcs\.revision=//p' | head -n 1)
        if [[ "$embedded" == "$build_sha" ]]; then
          pass "artifact:$name:embedded-source" "$embedded"
        else
          reject "embedded binary source revision mismatch: $name" "artifact:$name:embedded-source"
        fi
      else
        reject "cannot extract release binary from $name" "artifact:$name:embedded-source"
      fi
      rm -rf "$extract_dir"
    elif [[ "$require_local" == true ]]; then
      reject "go and tar are required for strict binary source verification" "artifact:$name:embedded-source"
    else
      warn "go/tar unavailable; embedded source revision not independently checked" "artifact:$name:embedded-source"
    fi
  elif [[ "$require_local" == true ]]; then
    reject "local asset missing: $name" "artifact:$name:file"
  else
    warn "local asset not downloaded: $name" "artifact:$name:file"
  fi

  sbom_file="$asset_dir/$(basename "$sbom")"
  expected=$(jq -r '.sbom.sha256' <<<"$artifact")
  if [[ -f "$sbom_file" ]]; then
    actual=$(sha256_file "$sbom_file")
    if [[ "$actual" == "$expected" ]]; then
      pass "artifact:$name:sbom:sha256" "$actual"
    else
      reject "hash mismatch: $sbom" "artifact:$name:sbom:sha256"
    fi
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

  if [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    pass "image:$kind:digest" "$digest"
  else
    reject "invalid image digest for $kind" "image:$kind:digest"
    continue
  fi

  if command -v docker >/dev/null 2>&1; then
    manifest_json=$(docker buildx imagetools inspect "$ref" --format '{{json .Manifest}}' 2>/dev/null || true)
    resolved=$(jq -r '.digest // empty' <<<"$manifest_json" 2>/dev/null || true)
    if [[ "$resolved" == "$digest" ]]; then
      pass "image:$kind:resolved-digest" "$resolved"
    elif [[ -n "$resolved" ]]; then
      reject "resolved image digest differs from receipt for $kind" "image:$kind:resolved-digest"
    elif [[ "$require_local" == true ]]; then
      reject "unable to resolve image reference: $ref" "image:$kind:resolved-digest"
    else
      warn "image reference not independently resolved: $ref" "image:$kind:resolved-digest"
    fi
  elif [[ "$require_local" == true ]]; then
    reject "docker/buildx is required for strict image digest verification" "image:$kind:resolved-digest"
  else
    warn "docker/buildx unavailable; image digest not independently resolved" "image:$kind:resolved-digest"
  fi

  while IFS= read -r platform; do
    platform_digest=$(jq -r '.digest // empty' <<<"$platform")
    if [[ "$platform_digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
      pass "image:$kind:platform:$platform_digest" "valid immutable digest"
    else
      reject "invalid platform digest for $kind" "image:$kind:platform"
    fi
  done < <(jq -c '.platforms[]?' <<<"$image")

  while IFS= read -r sbom; do
    name=$(jq -r '.name' <<<"$sbom")
    source=$(jq -r '.source' <<<"$sbom")
    image_digest=$(jq -r '.imageDigest' <<<"$sbom")
    expected=$(jq -r '.sha256' <<<"$sbom")
    sbom_file="$asset_dir/$(basename "$name")"

    if [[ "$source" == *@"$image_digest" && "$image_digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
      pass "image:$kind:sbom:$name:binding" "$source"
    else
      reject "SBOM source/digest mismatch: $name" "image:$kind:sbom:$name:binding"
    fi

    if [[ -f "$sbom_file" ]]; then
      actual=$(sha256_file "$sbom_file")
      if [[ "$actual" == "$expected" ]]; then
        pass "image:$kind:sbom:$name:sha256" "$actual"
      else
        reject "image SBOM hash mismatch: $name" "image:$kind:sbom:$name:sha256"
      fi
    elif [[ "$require_local" == true ]]; then
      reject "image SBOM missing: $sbom_file" "image:$kind:sbom:$name:file"
    else
      warn "image SBOM not downloaded: $name" "image:$kind:sbom:$name:file"
    fi
  done < <(jq -c '.sboms[]?' <<<"$image")
done < <(jq -c '.images[]' "$receipt")

finish

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

if [[ $# -ne 5 ]]; then
    echo "usage: $0 <version> <tag> <build-source-sha> <release-commit-sha> <cyclonedx-gomod-version>" >&2
    exit 2
fi

version=$1
tag=$2
build_source_sha=$3
release_commit_sha=$4
cyclonedx_version=$5
syft_version=${SYFT_VERSION:-v1.51.0}

operator_image_ref=${OPERATOR_IMAGE_REF:-${IMAGE_NAME:-}:$version}
bundle_image_ref=${BUNDLE_IMAGE_REF:-docker.io/testcamelk/camel-k-bundle:$version}

if [[ "$operator_image_ref" == :* ]]; then
    echo "IMAGE_NAME or OPERATOR_IMAGE_REF is required to bind the operator image" >&2
    exit 2
fi

for command in cyclonedx-gomod docker go jq tar; do
    command -v "$command" >/dev/null 2>&1 || {
        echo "required command not found: $command" >&2
        exit 1
    }
done

verify_syft() {
    command -v syft >/dev/null 2>&1 || {
        echo "required command not found: syft" >&2
        exit 1
    }

    local reported_version
    reported_version=$(syft --version | awk '{print $NF}')
    if [[ "$reported_version" != "$syft_version" && "v$reported_version" != "$syft_version" ]]; then
        echo "syft binary is not the required version ${syft_version}: got ${reported_version}" >&2
        exit 1
    fi
}

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        echo "required command not found: sha256sum or shasum" >&2
        exit 1
    fi
}

resolve_image() {
    local ref=$1
    local kind=$2
    local manifest_json=""
    local attempt

    for attempt in 1 2 3 4 5; do
        if manifest_json=$(docker buildx imagetools inspect "$ref" --format '{{json .Manifest}}' 2>/dev/null) && \
            jq -e '.digest | strings | startswith("sha256:")' >/dev/null <<<"$manifest_json"; then
            break
        fi
        manifest_json=""
        sleep 2
    done

    if [[ -z "$manifest_json" ]]; then
        echo "unable to resolve pushed $kind image manifest: $ref" >&2
        exit 1
    fi

    jq -cS \
        --arg kind "$kind" \
        --arg ref "$ref" \
        '{
            kind: $kind,
            ref: $ref,
            digest: .digest,
            mediaType: .mediaType,
            platforms: ((.manifests // [])
                | map({
                    digest: .digest,
                    mediaType: .mediaType,
                    platform: (.platform // null)
                })
                | sort_by([.platform.os // "", .platform.architecture // "", .platform.variant // "", .digest]))
        }' <<<"$manifest_json"
}

repository_without_tag() {
    local ref=${1%@*}
    local tail=${ref##*/}

    if [[ "$tail" == *:* ]]; then
        ref=${ref%:*}
    fi

    printf '%s\n' "$ref"
}

normalize_cyclonedx() {
    local input=$1
    local output=$2
    local source=$3
    local digest=$4

    jq -S \
        --arg source "$source" \
        --arg digest "$digest" \
        '
        del(.serialNumber, .metadata.timestamp)
        | .metadata.component.properties = ((.metadata.component.properties // []) + [
            {name: "org.apache.camel-k.assurance.image.source", value: $source},
            {name: "org.apache.camel-k.assurance.image.digest", value: $digest}
        ])
        ' "$input" > "$output"

    jq -e \
        --arg source "$source" \
        --arg digest "$digest" \
        '
        .bomFormat == "CycloneDX"
        and (.components | type == "array")
        and any(.metadata.component.properties[]?; .name == "org.apache.camel-k.assurance.image.source" and .value == $source)
        and any(.metadata.component.properties[]?; .name == "org.apache.camel-k.assurance.image.digest" and .value == $digest)
        ' "$output" >/dev/null
}

scan_image_sbom() {
    local repository=$1
    local digest=$2
    local output=$3
    local raw=$4

    SYFT_CHECK_FOR_APP_UPDATE=false syft scan \
        --from registry \
        "${repository}@${digest}" \
        --source-name "$repository" \
        --source-version "$version" \
        -q \
        -o "cyclonedx-json@1.6=${raw}"

    normalize_cyclonedx "$raw" "$output" "${repository}@${digest}" "$digest"
}

generate_image_sboms() {
    local image_json=$1
    local kind=$2
    local ref repository top_digest sbom_entries platform_count

    ref=$(jq -r '.ref' <<<"$image_json")
    repository=$(repository_without_tag "$ref")
    top_digest=$(jq -r '.digest' <<<"$image_json")
    sbom_entries="$workdir/${kind}.sboms.jsonl"
    : > "$sbom_entries"

    platform_count=$(jq '[.platforms[]? | select(
        .digest != null and
        .platform != null and
        .platform.os != null and
        .platform.architecture != null and
        .platform.os != "unknown" and
        .platform.architecture != "unknown"
    )] | length' <<<"$image_json")

    if [[ "$platform_count" -gt 0 ]]; then
        while IFS=$'\t' read -r digest os architecture variant; do
            local suffix output raw sbom_sha immutable_ref
            suffix="${os}-${architecture}"
            if [[ -n "$variant" && "$variant" != "null" ]]; then
                suffix="${suffix}-${variant//\//-}"
            fi
            output="camel-k-image-${kind}-${suffix}.sbom.cdx.json"
            raw="$workdir/${kind}-${suffix}.raw.cdx.json"
            immutable_ref="${repository}@${digest}"

            scan_image_sbom "$repository" "$digest" "$output" "$raw"
            sbom_sha=$(sha256_file "$output")

            jq -cn \
                --arg name "$output" \
                --arg source "$immutable_ref" \
                --arg digest "$digest" \
                --arg os "$os" \
                --arg architecture "$architecture" \
                --arg variant "$variant" \
                --arg sha256 "$sbom_sha" \
                '{
                    name: $name,
                    format: "CycloneDX JSON 1.6",
                    source: $source,
                    imageDigest: $digest,
                    platform: {
                        os: $os,
                        architecture: $architecture,
                        variant: (if $variant == "" or $variant == "null" then null else $variant end)
                    },
                    sha256: $sha256
                }' >> "$sbom_entries"
        done < <(jq -r '.platforms[]? | select(
            .digest != null and
            .platform != null and
            .platform.os != null and
            .platform.architecture != null and
            .platform.os != "unknown" and
            .platform.architecture != "unknown"
        ) | [.digest, .platform.os, .platform.architecture, (.platform.variant // "")] | @tsv' <<<"$image_json")
    else
        local output raw sbom_sha immutable_ref
        output="camel-k-image-${kind}.sbom.cdx.json"
        raw="$workdir/${kind}.raw.cdx.json"
        immutable_ref="${repository}@${top_digest}"

        scan_image_sbom "$repository" "$top_digest" "$output" "$raw"
        sbom_sha=$(sha256_file "$output")

        jq -cn \
            --arg name "$output" \
            --arg source "$immutable_ref" \
            --arg digest "$top_digest" \
            --arg sha256 "$sbom_sha" \
            '{
                name: $name,
                format: "CycloneDX JSON 1.6",
                source: $source,
                imageDigest: $digest,
                platform: null,
                sha256: $sha256
            }' >> "$sbom_entries"
    fi

    jq -s '.' "$sbom_entries"
}

verify_syft

workdir=$(mktemp -d)
entries="$workdir/artifacts.jsonl"
trap 'rm -rf "$workdir"' EXIT

shopt -s nullglob
archives=(camel-k-client-"$version"-*.tar.gz)
shopt -u nullglob

if [[ ${#archives[@]} -eq 0 ]]; then
    echo "no Camel K client archives found for version $version" >&2
    exit 1
fi

IFS=$'\n' archives=($(printf '%s\n' "${archives[@]}" | sort))
unset IFS

for archive in "${archives[@]}"; do
    target=${archive#camel-k-client-"$version"-}
    target=${target%.tar.gz}

    case "$target" in
        windows-*) binary_name=kamel.exe ;;
        *) binary_name=kamel ;;
    esac

    extract_dir="$workdir/$target"
    mkdir -p "$extract_dir"
    tar -xzf "$archive" -C "$extract_dir" "$binary_name"
    chmod +x "$extract_dir/$binary_name"

    sbom="${archive%.tar.gz}.sbom.cdx.json"
    cyclonedx-gomod bin \
        -json \
        -noserial \
        -notimestamp \
        -version "$version" \
        -output "$sbom" \
        "$extract_dir/$binary_name"

    binary_source_sha=$(go version -m "$extract_dir/$binary_name" | sed -n 's/^[[:space:]]*build[[:space:]]*vcs\.revision=//p' | head -n 1)
    if [[ -z "$binary_source_sha" ]]; then
        echo "binary source revision missing for $archive" >&2
        exit 1
    fi
    if [[ "$binary_source_sha" != "$build_source_sha" ]]; then
        echo "binary source revision mismatch for $archive: expected $build_source_sha, got $binary_source_sha" >&2
        exit 1
    fi

    archive_sha=$(sha256_file "$archive")
    sbom_sha=$(sha256_file "$sbom")

    jq -cn \
        --arg name "$archive" \
        --arg target "$target" \
        --arg sha256 "$archive_sha" \
        --arg binarySourceSha "$binary_source_sha" \
        --arg sbomName "$sbom" \
        --arg sbomSha256 "$sbom_sha" \
        '{
            name: $name,
            target: $target,
            sha256: $sha256,
            binarySourceSha: $binarySourceSha,
            sbom: {
                name: $sbomName,
                format: "CycloneDX JSON",
                mode: "binary",
                sha256: $sbomSha256
            }
        }' >> "$entries"
done

module_sbom=null
if [[ -f sbom.json ]]; then
    module_sbom=$(jq -cn \
        --arg name "sbom.json" \
        --arg sha256 "$(sha256_file sbom.json)" \
        '{name: $name, format: "CycloneDX JSON", mode: "module", sha256: $sha256}')
fi

operator_image=$(resolve_image "$operator_image_ref" "operator")
bundle_image=$(resolve_image "$bundle_image_ref" "olm-bundle")
operator_sboms=$(generate_image_sboms "$operator_image" "operator")
bundle_sboms=$(generate_image_sboms "$bundle_image" "olm-bundle")
operator_image=$(jq -cS --argjson sboms "$operator_sboms" '. + {sboms: $sboms}' <<<"$operator_image")
bundle_image=$(jq -cS --argjson sboms "$bundle_sboms" '. + {sboms: $sboms}' <<<"$bundle_image")
buildx_version=$(docker buildx version | head -n 1)
syft_build=$syft_version

jq -s '.' "$entries" > "$workdir/artifacts.json"

jq -Sn \
    --argjson schemaVersion 1 \
    --arg version "$version" \
    --arg tag "$tag" \
    --arg buildSourceSha "$build_source_sha" \
    --arg releaseCommitSha "$release_commit_sha" \
    --arg generatorModule "github.com/CycloneDX/cyclonedx-gomod" \
    --arg generatorVersion "$cyclonedx_version" \
    --arg imageSbomGenerator "github.com/anchore/syft" \
    --arg imageSbomGeneratorVersion "$syft_build" \
    --arg registryResolver "$buildx_version" \
    --slurpfile artifacts "$workdir/artifacts.json" \
    --argjson moduleSbom "$module_sbom" \
    --argjson operatorImage "$operator_image" \
    --argjson bundleImage "$bundle_image" \
    '{
        schemaVersion: $schemaVersion,
        release: {
            version: $version,
            tag: $tag,
            buildSourceSha: $buildSourceSha,
            releaseCommitSha: $releaseCommitSha
        },
        generator: {
            module: $generatorModule,
            version: $generatorVersion,
            imageSbom: {
                module: $imageSbomGenerator,
                version: $imageSbomGeneratorVersion
            },
            registryResolver: $registryResolver
        },
        moduleSbom: $moduleSbom,
        artifacts: $artifacts[0],
        images: [$operatorImage, $bundleImage]
    }' > release-assurance.json

printf '%s  %s\n' "$(sha256_file release-assurance.json)" "release-assurance.json" > release-assurance.json.sha256

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

#
# Use bash explicitly in this Makefile to avoid unexpected platform
# incompatibilities among Linux distros.
#
SHELL := /bin/bash

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
LOCALBIN := $(PROJECT_PATH)/bin

VERSIONFILE := pkg/util/defaults/defaults.go
VERSION ?= 2.5.0-SNAPSHOT
LAST_RELEASED_IMAGE_NAME := camel-k-operator
LAST_RELEASED_VERSION ?= 2.4.0
DEFAULT_RUNTIME_VERSION := 3.8.1
CONTROLLER_GEN_VERSION := v0.15.0
CODEGEN_VERSION := v0.29.7
OPERATOR_SDK_VERSION := v1.30.0
KUSTOMIZE_VERSION := v4.5.4
OPM_VERSION := v1.24.0
LINTER_VERSION ?= v1.59.1
GOVULNCHECK_VERSION ?= latest

LINTER ?= $(LOCALBIN)/golangci-lint
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
GOIMPORT ?= $(LOCALBIN)/goimports

BASE_IMAGE := eclipse-temurin:17
LOCAL_REPOSITORY := /etc/maven/m2
IMAGE_NAME ?= docker.io/apache/camel-k
# Test for arm64, fall back to amd64
IMAGE_ARCH ?= $(if $(filter arm64 aarch64,$(shell uname -m)),arm64,amd64)
#
# Situations when user wants to override
# the image name and version
# - used in kustomize install
# - used in making bundle
# - need to preserve original image and version as used in other files
#
CUSTOM_IMAGE ?= $(IMAGE_NAME)
CUSTOM_VERSION ?= $(VERSION)
OPERATOR_VERSION := $(subst -SNAPSHOT,,$(CUSTOM_VERSION))

METADATA_IMAGE_NAME := $(CUSTOM_IMAGE)-metadata
BUNDLE_IMAGE_NAME ?= $(CUSTOM_IMAGE)-bundle
RELEASE_GIT_REMOTE := origin
GIT_COMMIT := $(shell if [ -d .git ]; then git rev-list -1 HEAD; else echo "$(CUSTOM_VERSION)"; fi)
LINT_GOGC ?= 20
LINT_DEADLINE ?= 10m
DEBUG_MODE ?= false
OPENSHIFT_REGISTRY := image-registry.openshift-image-registry.svc:5000

# olm bundle vars
MANAGER := pkg/resources/config/manager
MANIFESTS := pkg/resources/config/manifests
DEFAULT_CHANNEL ?= $(shell echo "stable-v$(word 1,$(subst ., ,$(lastword $(OPERATOR_VERSION))))")
CHANNELS ?= $(DEFAULT_CHANNEL),latest
PACKAGE := camel-k
CSV_VERSION := $(OPERATOR_VERSION)
CSV_NAME := $(PACKAGE).v$(CSV_VERSION)
# Final CSV name that replaces the name required by the operator-sdk
# Has to be replaced after the bundle has been generated
CSV_PRODUCTION_NAME := $(LAST_RELEASED_IMAGE_NAME).v$(CSV_VERSION)
CSV_DISPLAY_NAME := Camel K Operator
CSV_SUPPORT := Camel
CSV_REPLACES := $(LAST_RELEASED_IMAGE_NAME).v$(LAST_RELEASED_VERSION)
CSV_FILENAME := $(PACKAGE).clusterserviceversion.yaml
CSV_PATH := $(MANIFESTS)/bases/$(CSV_FILENAME)
CSV_PRODUCTION_PATH := bundle/manifests/$(CSV_FILENAME)

# Test Bundle Index
BUNDLE_INDEX := quay.io/operatorhubio/catalog:latest
INDEX_DIR := index
OPM := opm

# Used to push pre-release artifacts
STAGING_IMAGE := docker.io/camelk/camel-k
INSTALL_DEFAULT_KAMELETS ?= true

# When performing integration tests, it is not necessary to always execute build, especially
# in e2e tests when lots of tests are being executed sequentially & the build has already taken place.
# In this case, if DO_TEST_PREBUILD is set to false then TEST_PREBUILD will be set to blank
# and those rules with a dependency on the build rule will no longer call it (see do-build)
DO_TEST_PREBUILD ?= true
TEST_PREBUILD = build
# Tests may run in parallel to each other. This count sets the amount of tests run in parallel.
# (default value would be otherwise GOMAXPROCS)
TEST_COMMON_PARALLEL_COUNT ?= 2
TEST_ADVANCED_PARALLEL_COUNT ?= 2

# OLM (Operator Lifecycle Manager and Operator Hub): uncomment to override operator settings at build time
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultOperatorName=camel-k-operator
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultPackage=camel-k
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultChannel=alpha
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultSource=community-operators
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultSourceNamespace=openshift-marketplace
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultStartingCSV=
#GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/olm.DefaultGlobalNamespace=openshift-operators

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Build
ifdef GIT_COMMIT
GOLDFLAGS += -X github.com/apache/camel-k/v2/pkg/util/defaults.GitCommit=$(GIT_COMMIT)
else
$(warning Could not retrieve a valid Git Commit)
endif

GOFLAGS = -ldflags "$(GOLDFLAGS)" -trimpath

ifeq ($(DEBUG_MODE),true)
GOFLAGS += -gcflags="all=-N -l"
endif

define LICENSE_HEADER
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
endef

export LICENSE_HEADER
default: build

update-docs: build-resources
	./script/update_docs.sh

update-default-camel:
	./script/update_default_camel.sh

bump-replace:
	@# Bump version and replace with the variables provided by the user
	@sed -i 's/^VERSION ?= .*$//VERSION ?= $(VERSION)/' ./script/Makefile
	@mvn -f java/crds/pom.xml versions:set -DgenerateBackupPoms=false -DnewVersion=$(VERSION)
	@mvn -f java/maven-logging/pom.xml versions:set -DgenerateBackupPoms=false -DnewVersion=$(VERSION)
	@sed -i 's/^LAST_RELEASED_VERSION ?= .*$//LAST_RELEASED_VERSION ?= $(LAST_RELEASED_VERSION)/' ./script/Makefile

bump: bump-replace codegen bundle

# Generates the version file
codegen:
	$(info Regenerating $(VERSIONFILE))
	@echo "/*" > $(VERSIONFILE)
	@echo "$$LICENSE_HEADER" >> $(VERSIONFILE)
	@echo "*/" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "package defaults" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "// ***********************" >> $(VERSIONFILE)
	@echo "//  DO NOT EDIT THIS FILE"  >> $(VERSIONFILE)
	@echo "// ***********************" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "const (" >> $(VERSIONFILE)
	@echo "  // Version -- " >> $(VERSIONFILE)
	@echo "  Version = \"$(CUSTOM_VERSION)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // DefaultRuntimeVersion -- " >> $(VERSIONFILE)
	@echo "  DefaultRuntimeVersion = \"$(DEFAULT_RUNTIME_VERSION)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // baseImage -- " >> $(VERSIONFILE)
	@echo "  baseImage = \"$(BASE_IMAGE)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // LocalRepository -- " >> $(VERSIONFILE)
	@echo "  LocalRepository = \"$(LOCAL_REPOSITORY)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // ImageName -- " >> $(VERSIONFILE)
	@echo "  ImageName = \"$(CUSTOM_IMAGE)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // OpenShiftRegistryAddress -- " >> $(VERSIONFILE)
	@echo "  OpenShiftRegistryAddress = \"$(OPENSHIFT_REGISTRY)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // installDefaultKamelets -- " >> $(VERSIONFILE)
	@echo "  installDefaultKamelets = $(INSTALL_DEFAULT_KAMELETS)" >> $(VERSIONFILE)
	@echo ")" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "// GitCommit must be provided during application build" >> $(VERSIONFILE)
	@echo "var GitCommit string" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)

	gofmt -w pkg/util/defaults/defaults.go

generate: codegen-tools-install
	@# We better force the developer to generate all the resources everytime as
	@# it happens we tend to forget some code generation in the past
	./script/gen_client.sh
	./script/gen_crd.sh
	./script/gen_doc.sh
	cd pkg/apis/camel && $(CONTROLLER_GEN) paths="./..." object
	cd addons/keda/duck && $(CONTROLLER_GEN) paths="./..." object
	cd addons/strimzi/duck && $(CONTROLLER_GEN) paths="./..." object
	./script/gen_client_strimzi.sh

codegen-tools-install: controller-gen
	@# We must force the installation to make sure we are using the correct version
	@# Note: as there is no --version in the tools, we cannot rely on cached local versions
	@echo "Installing k8s.io/code-generator tools with version $(CODEGEN_VERSION)"
	go install k8s.io/code-generator/cmd/applyconfiguration-gen@$(CODEGEN_VERSION)
	go install k8s.io/code-generator/cmd/client-gen@$(CODEGEN_VERSION)
	go install k8s.io/code-generator/cmd/lister-gen@$(CODEGEN_VERSION)
	go install k8s.io/code-generator/cmd/informer-gen@$(CODEGEN_VERSION)

build: codegen build-resources test build-kamel build-compile-integration-tests

build-platform: build build-kamel-platform

ci-build: clean codegen set-version check-licenses dir-licenses build-kamel cross-compile

check_jdk_version:
	@go run ./cmd/util/check_jdk_version

do-build: gotestfmt-install check_jdk_version
ifeq ($(DO_TEST_PREBUILD),true)
TEST_PREBUILD = build
else
TEST_PREBUILD =
endif

gotestfmt-install:
ifeq (, $(shell command -v gotestfmt 2> /dev/null))
	go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
endif

# Build images without running unit tests
# NOTEST=1 make images
test: do-build
ifndef NOTEST
	@echo "####### Running unit test..."
	go test ./... $(COVERAGE_OPTS)
else
	@echo "####### Skipping unit test..."
endif

#
# Setup the Knative test environment
#
setup-knative:
	./e2e/knative/files/setup.sh

#
# Common tests that do not require any customized operator setting. They can leverage a unique namespaced operator installation to reduce
# the time to complete (they are used also as smoke test for nightly release)
#
test-common:
	FAILED=0; \
	go test -p $(TEST_COMMON_PARALLEL_COUNT) -timeout 90m -v ./e2e/common/... --parallel 1 -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Smoke tests are a subset of common tests used to quickly verify the operator
#
test-smoke:
	FAILED=0; \
	go test -p $(TEST_COMMON_PARALLEL_COUNT) -timeout 30m -v \
		./e2e/common/languages \
		./e2e/common/traits \
		--parallel 1 \
	  -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Common tests that require some particular operator setting or need to be installed in multiple namespaces
#
test-advanced:
	FAILED=0; \
	go test -p 1 -timeout 90m -v ./e2e/advanced --parallel $(TEST_ADVANCED_PARALLEL_COUNT) -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Installation and upgrade procedure test using
#
test-install:
	FAILED=0; \
	go test -timeout 20m -v ./e2e/install/kustomize -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	go test -timeout 20m -v ./e2e/install/helm -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	go test -timeout 60m -v ./e2e/install/upgrade -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Installation procedure test via OLM
#
test-install-olm:
	FAILED=0; \
	go test -timeout 20m -v ./e2e/install/olm -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Knative tests that require the presence of Knative configuration
#
test-knative:
	FAILED=0; \
	go test -timeout 60m -v ./e2e/knative -tags=integration $(GOTESTFMT) || ((FAILED++)); \
	exit $${FAILED}

#
# Telemetry tests that require the configuration of telemetry endpoints
#
test-telemetry:
	go test -timeout 30m -v ./e2e/telemetry -tags=integration $(GOTESTFMT)

#
# Quarkus native test (requires certain CPU and memory conditions)
#
test-quarkus-native:
	go test -timeout 180m -v ./e2e/native -tags=integration $(GOTESTFMT)

build-kamel:
	@echo "####### Building kamel CLI for linux/$(IMAGE_ARCH) architecture..."
	CGO_ENABLED=0 GOOS=linux GOARCH=$(IMAGE_ARCH) go build $(GOFLAGS) -o build/_output/bin/kamel-$(IMAGE_ARCH) ./cmd/kamel/*.go
	# Symbolic link to a local CLI
	ln -sf build/_output/bin/kamel-$(IMAGE_ARCH) ./kamel

build-kamel-platform:
	# Perform only when running on OS other than linux
ifneq ($(shell uname -s 2>/dev/null || echo Unknown),Linux)
	@echo "####### Building platform specific kamel CLI for $(IMAGE_ARCH) architecture..."
	CGO_ENABLED=0 GOARCH=$(IMAGE_ARCH) go build $(GOFLAGS) -o build/_output/bin/kamel-$(IMAGE_ARCH) ./cmd/kamel/*.go
	# Symbolic link to a local CLI
	ln -sf build/_output/bin/kamel-$(IMAGE_ARCH) ./kamel
endif

build-resources:
	./script/get_catalog.sh $(DEFAULT_RUNTIME_VERSION)

build-compile-integration-tests:
	@echo "####### Compiling integration tests..."
	export CAMEL_K_E2E_JUST_COMPILE="true"; \
	go test -run nope -tags="integration" ./e2e/...

clean:
	# disable gomodules when executing go clean:
	#
	#    https://github.com/golang/go/issues/31002
	#
	GO111MODULE=off go clean
	rm -f camel-k
	rm -f kamel
	rm -f *.test
	rm -rf $(LOCALBIN)
	rm -rf build/_maven_output
	rm -rf build/_maven_overlay
	rm -rf build/_output
	rm -rf camel-k-client-*.tar.gz
	rm -rf camel-k-examples-*.tar.gz
	rm -rf bundle
	rm -rf $(INDEX_DIR) $(INDEX_DIR).Dockerfile

dep:
	@echo "Cleaning dependencies..."
	go mod tidy

detect-os:
ifeq '$(findstring ;,$(PATH))' ';'
OS := Windows
OS_LOWER := windows
else
OS := $(shell echo $$(uname 2>/dev/null) || echo Unknown)
OS := $(patsubst CYGWIN%,Cygwin,$(OS))
OS := $(patsubst MSYS%,MSYS,$(OS))
OS := $(patsubst MINGW%,MSYS,$(OS))
OS_LOWER := $(shell echo $(OS) | tr '[:upper:]' '[:lower:]')
endif

check: lint vuln

.PHONY: lint
lint: golangci-lint
	GOGC=$(LINT_GOGC) $(LINTER) run --config .golangci.yml --out-format colored-tab --timeout $(LINT_DEADLINE) --verbose

.PHONY: lint-fix
lint-fix: golangci-lint
	GOGC=$(LINT_GOGC) $(LINTER) run --config .golangci.yml --out-format colored-tab --timeout $(LINT_DEADLINE) --fix

.PHONY: vuln
vuln: govulncheck
	@$(GOVULNCHECK) ./...

.PHONY: fmt
fmt: goimport
	$(GOIMPORT) -l -w .
	go fmt ./...

dir-licenses:
	./script/vendor-license-directory.sh

check-licenses:
	./script/check_licenses.sh

# The below are dependencies needed for maven structured logs. We must bundle into the final container image.
maven-overlay:
	@echo "####### Preparing maven dependencies bundle..."
	rm -rf build/_maven_overlay
	mkdir -p build/_maven_overlay
	./script/maven_overlay.sh build/_maven_overlay

TARGET_STAGE := base
ifeq ($(DEBUG_MODE),true)
	TARGET_STAGE := debug
	CUSTOM_IMAGE := $(CUSTOM_IMAGE)-debug
endif

DOCKER_TAG := $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-$(IMAGE_ARCH)

images: build maven-overlay image-build build-kamel-platform

image-build:
ifneq (,$(findstring SNAPSHOT,$(DEFAULT_RUNTIME_VERSION)))
	./script/package_maven_artifacts.sh -d "$(CAMEL_K_RUNTIME_DIR)" $(DEFAULT_RUNTIME_VERSION)
endif
	@echo "####### Building Camel K operator arch $(IMAGE_ARCH) container image..."
	mkdir -p build/_maven_output
	docker buildx build --target $(TARGET_STAGE) --platform=linux/$(IMAGE_ARCH) --build-arg IMAGE_ARCH=$(IMAGE_ARCH) --load -t $(DOCKER_TAG) -f build/Dockerfile .
	docker tag $(DOCKER_TAG) $(CUSTOM_IMAGE):$(CUSTOM_VERSION)

# Mainly used for internal CI purposes
image-push:
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)

# Make sure the current docker builder must supports the wanted platform list, which may not be the case for the default builder
#
# docker buildx inspect
# ...
# Platforms: linux/amd64*, linux/arm64*
#
#
# docker buildx create --name mybuilder --platform linux/amd64,linux/arm64
# docker buildx use mybuilder
images-all:
	make IMAGE_ARCH=arm64 images
	make IMAGE_ARCH=amd64 images

images-push:
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-amd64
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)
	@if docker inspect $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 &> /dev/null; then \
		echo "Image $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 exists, building the multiarch manifest"; \
		docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64; \
		docker manifest create $(CUSTOM_IMAGE):$(CUSTOM_VERSION) --amend $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-amd64 --amend $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64; \
		docker manifest push --purge $(CUSTOM_IMAGE):$(CUSTOM_VERSION); \
	else \
		echo "Image $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 does not exist"; \
	fi

images-push-staging:
	docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-amd64 $(STAGING_IMAGE):$(CUSTOM_VERSION)-amd64
	docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-amd64 $(STAGING_IMAGE):$(CUSTOM_VERSION)
	docker push $(STAGING_IMAGE):$(CUSTOM_VERSION)-amd64
	docker push $(STAGING_IMAGE):$(CUSTOM_VERSION)
	@if docker inspect $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 &> /dev/null; then \
		echo "Image $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 exists, building the multiarch manifest"; \
		docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 $(STAGING_IMAGE):$(CUSTOM_VERSION)-arm64; \
		docker push $(STAGING_IMAGE):$(CUSTOM_VERSION)-arm64; \
		docker manifest create $(STAGING_IMAGE):$(CUSTOM_VERSION) --amend $(STAGING_IMAGE):$(CUSTOM_VERSION)-amd64 --amend $(STAGING_IMAGE):$(CUSTOM_VERSION)-arm64; \
		docker manifest push --purge $(STAGING_IMAGE):$(CUSTOM_VERSION); \
	else \
		echo "Image $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 does not exist"; \
	fi

get-image:
	@echo $(CUSTOM_IMAGE)

get-version:
	@echo $(CUSTOM_VERSION)

get-last-released-img-name:
	@echo $(LAST_RELEASED_IMAGE_NAME)

get-last-released-version:
	@echo $(LAST_RELEASED_VERSION)

get-csv-name:
	@echo $(CSV_PRODUCTION_NAME)

get-bundle-image:
	@echo $(BUNDLE_IMAGE_NAME)

set-version:
	./script/set_version.sh $(CUSTOM_VERSION) $(CUSTOM_IMAGE)

git-tag:
	./script/git_tag.sh $(CUSTOM_VERSION) $(RELEASE_GIT_REMOTE)

cross-compile:
	# we must wrap the goldflags parameters with quotes as they will need to
	# be processed as a single argument by the cross compile script
	./script/cross_compile.sh $(CUSTOM_VERSION) $(subst ","\",$(GOFLAGS))

release: clean codegen set-version build-resources check-licenses images images-push cross-compile git-tag

release-staging: clean codegen set-version build-resources check-licenses images-all images-push-staging cross-compile git-tag sbomgen

sbomgen:
	cyclonedx-gomod mod -licenses -json -output sbom.json

# Build images
images-nightly: clean codegen set-version build-resources images-all

# Release artifacts
release-nightly: cross-compile images-push

release-helm:
	./script/release_helm.sh ${VERSION}

release-kustomize:
	RELEASE_VERSION=$(CUSTOM_VERSION) \
	RELEASE_NAME=$(PACKAGE) \
	./script/release_kustomize.sh

.PHONY: do-build build build-kamel build-kamel-platform build-resources dep codegen images images-push images-push-staging image-build test check clean release cross-compile package-examples set-version git-tag check-licenses build-resources release-helm release-staging release-nightly get-staging-repo get-version
.PHONY: controller-gen kubectl kustomize operator-sdk opm

# find or download controller-gen if necessary
controller-gen:
ifeq (, $(shell command -v controller-gen 2> /dev/null))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell command -v controller-gen 2> /dev/null)
endif

kubectl:
ifeq (, $(shell command -v kubectl 2> /dev/null))
	$(error "No kubectl found in PATH. Please install and re-run")
endif

kustomize:
ifeq (, $(shell command -v kustomize 2> /dev/null))
	go install sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION)
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell command -v kustomize 2> /dev/null)
endif

operator-sdk: detect-os
	@echo "####### Installing operator-sdk version $(OPERATOR_SDK_VERSION)..."
	curl \
		-s -L https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$(OS_LOWER)_amd64 \
		-o operator-sdk ; \
	chmod +x operator-sdk ;\
	mkdir -p $(GOBIN) ;\
	mv operator-sdk $(GOBIN)/ ;
OPERATOR_SDK=$(GOBIN)/operator-sdk

opm: detect-os
ifeq (, $(shell command -v opm 2> /dev/null))
	@{ \
	set -e ;\
	curl \
		-L https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$(OS_LOWER)-amd64-opm \
		-o opm; \
	chmod +x opm;\
	mkdir -p $(GOBIN) ;\
	mv opm $(GOBIN)/ ;\
	}
OPM=$(GOBIN)/opm
else
	@{ \
	echo -n "opm already installed: "; \
  opm version | sed -n 's/.*"v\([^"]*\)".*/\1/p'; \
	echo " If this is less than $(OPM_VERSION) then please consider moving it aside and allowing the approved version to be downloaded."; \
	}
OPM=$(shell command -v opm 2> /dev/null)
endif

yq:
ifeq (, $(shell command -v yq 2> /dev/null))
	@GO111MODULE=on go install github.com/mikefarah/yq/v3
YQ=$(GOBIN)/yq
else
YQ=$(shell command -v yq 2> /dev/null)
endif

.PHONY: $(BUNDLE_CAMEL_APIS) pre-bundle bundle bundle-build

# - Have to copy pkg/apis since it is a module in its own right
# - The dependency of v1alpha1 api on v1 results in duplication
#   of the v1 schema in those CRDs so need to split them up
# - No way to add a licence header to the CRD other then post-script
BUNDLE_CAMEL_APIS = v1 v1alpha1

$(BUNDLE_CAMEL_APIS): operator-sdk
	@# Clean up api directory, copy over fresh version
	rm -rf api_$@ && cp -rf pkg/apis api_$@
	@# Remove the camel directory and re-copy only the required api
	rm -rf api_$@/camel/* && cp -rf pkg/apis/camel/$@ api_$@/camel/
	@# operator-sdk generate ... cannot execute across separate modules so need to temporarily move api
	$(OPERATOR_SDK) generate kustomize manifests --apis-dir $(addprefix api_, $@) -q --input-dir $(MANIFESTS) --output-dir $(MANIFESTS)
	@# Adds the licence header to the csv file.
	./script/add_license.sh $(MANIFESTS)/bases ./script/headers/yaml.txt
	./script/add_createdAt.sh $(MANIFESTS)/bases
	@# Clean up temporary working api directories
	rm -rf api_*

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
ifneq ($(origin PACKAGE), undefined)
BUNDLE_PACKAGE := --package=$(PACKAGE)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL) $(BUNDLE_PACKAGE)
# TODO automate these configuration retrieving the content from go.mod
BUNDLE_METADATA_OPENSHIFT_VERSION ?= "v4.14"
MIN_KUBE_VERSION ?= "1.24.0"

#
# Tailor the manifest according to default values for this project
# Note. to successfully make the bundle the name must match that specified in the PROJECT file
#
pre-bundle:
# bundle name must match that which appears in PROJECT file
	@sed -i 's/projectName: .*/projectName: $(PACKAGE)/' PROJECT
	@sed -i 's~^    containerImage: .*~    containerImage: $(CUSTOM_IMAGE):$(CUSTOM_VERSION)~' $(CSV_PATH)
	@sed -i 's/^    support: .*/    support: $(CSV_SUPPORT)/' $(CSV_PATH)
	@sed -i 's/^  name: .*.\(v.*\)/  name: $(CSV_NAME)/' $(CSV_PATH)
	@sed -i 's/^  displayName: .*/  displayName: $(CSV_DISPLAY_NAME)/' $(CSV_PATH)
	@sed -i 's/^  version: .*/  version: $(CSV_VERSION)/' $(CSV_PATH)
	@sed -i 's/^  minKubeVersion: .*/  minKubeVersion: $(MIN_KUBE_VERSION)/' $(CSV_PATH)
	@if grep -q replaces $(CSV_PATH); \
		then sed -i 's/^  replaces: .*/  replaces: $(CSV_REPLACES)/' $(CSV_PATH); \
		else sed -i '/  version: ${CSV_VERSION}/a \ \ replaces: $(CSV_REPLACES)' $(CSV_PATH); \
	fi

bundle: set-version kustomize operator-sdk pre-bundle $(BUNDLE_CAMEL_APIS)
	@# Display BUNDLE_METADATA_OPTS for debugging
	$(info BUNDLE_METADATA_OPTS=$(BUNDLE_METADATA_OPTS))
	@# Sets the operator image to the preferred image:tag
	@cd pkg/resources/config/manifests && $(KUSTOMIZE) edit set image $(IMAGE_NAME)=$(CUSTOM_IMAGE):$(CUSTOM_VERSION)
	@# Build kustomize manifests
	@$(KUSTOMIZE) build pkg/resources/config/manifests | \
		$(OPERATOR_SDK) generate bundle \
			-q --overwrite --version $(OPERATOR_VERSION) \
			--kustomize-dir pkg/resources/config/manifests $(BUNDLE_METADATA_OPTS)
	@# Move the dockerfile into the bundle directory
	@mv bundle.Dockerfile bundle/Dockerfile && sed -i 's/bundle\///g' bundle/Dockerfile
	@# Adds the licence headers to the csv file
	./script/add_license.sh bundle/manifests ./script/headers/yaml.txt
# Rename the CSV name to conform with the existing released operator versions
# This cannot happen in pre-bundle as the operator-sdk generation expects a CSV name the same as PACKAGE
	@sed -i "s/^  name: $(CSV_NAME)/  name: $(CSV_PRODUCTION_NAME)/" $(CSV_PRODUCTION_PATH)
	@echo "  com.redhat.openshift.versions: $(BUNDLE_METADATA_OPENSHIFT_VERSION)" >> ./bundle/metadata/annotations.yaml
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build: bundle
	cd bundle && docker build -f Dockerfile -t $(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION) .

.PHONY: bundle-push bundle-index

bundle-push: bundle-build
	docker push $(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION)

bundle-test: operator-sdk
ifndef NAMESPACE
	@echo "WARN: no namespace specified, using default namespace $(DEFAULT_NS)"
	$(eval NAMESPACE=$(DEFAULT_NS))
endif
	$(OPERATOR_SDK) run bundle $(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION) -n $(NAMESPACE)

#
# Builds a test catalog index for installing the operator via an OLM
#
bundle-index: opm yq
	BUNDLE_INDEX=$(BUNDLE_INDEX) INDEX_DIR=$(INDEX_DIR) PACKAGE=$(PACKAGE) YQ=$(YQ) \
	OPM=$(OPM) BUNDLE_IMAGE=$(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION) CSV_NAME=$(CSV_PRODUCTION_NAME) \
	CSV_SKIPS=$(CSV_SKIP_RANGE) CSV_REPLACES=$(CSV_REPLACES) CHANNELS="$(CHANNELS)" \
	./script/build_bundle_index.sh


## Location to install dependencies to
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: golangci-lint
golangci-lint: $(LINTER)
$(LINTER): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINTER_VERSION)

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK)
$(GOVULNCHECK): $(LOCALBIN)
	@test -s $(GOVULNCHECK) || \
	GOBIN=$(LOCALBIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: goimport
goimport: $(GOIMPORT)
$(GOIMPORT): $(LOCALBIN)
	@test -s $(LOCALBIN)/goimport || \
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

#####
# START Local installation procedure. Handy for development purpose
#####

KUSTOMIZE_DIR = "install/overlays/kubernetes/descoped"
DEFAULT_NS = "camel-k"

.PHONY: install install-k8s-global install-k8s-ns install-openshift-global install-openshift-ns

clone-kustomize-dir:
	$(eval KUST_TMP=$(shell mktemp -d))
	@echo "... cloning installation files to $(KUST_TMP)"
	cp -Lr install $(KUST_TMP)/.

# the set-operator targets are mostly thought as an utility for the e2e which require certain operator settings
set-operator-id:
ifdef OPERATOR_ID
	@echo "INFO: setting operator ID to $(OPERATOR_ID)"
	@sed -i 's/name: .*/name: $(OPERATOR_ID)/' $(KUST_TMP)/install/overlays/platform/integration-platform.yaml
	cd $(KUST_TMP)/install/overlays/platform/ && kustomize edit add annotation camel.apache.org/operator.id:$(OPERATOR_ID)
	@sed -i 's/value: .*/value: $(OPERATOR_ID)/' $(KUST_TMP)/$(KUSTOMIZE_DIR)/patch-operator-id.yaml
	@echo "INFO: adding a suffix for ClusterRoleBindings $(OPERATOR_ID)"
	cd $(KUST_TMP)/install/base/config/rbac/descoped && kustomize edit set namesuffix -- -$(OPERATOR_ID)
	cd $(KUST_TMP)/install/base/config/rbac/openshift/descoped && kustomize edit set namesuffix -- -$(OPERATOR_ID)
else
	@echo "INFO: no operator ID variable \"OPERATOR_ID\" provided. Keeping default."
endif

# ENV is expected to be in the formal ENV="myvar1=v1 myvar2=v2 ..."
set-operator-env:
ifdef ENV
	@for n in $(ENV); do \
		k=$${n%=*}; \
		v=$${n#*=}; \
		echo "INFO: setting operator env var $${k}=$${v}"; \
		cd $(KUST_TMP)/$(KUSTOMIZE_DIR) && kustomize edit add patch --kind Deployment --patch \
		 	"[{\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/env/-\", \"value\": {\"name\": \"$${k}\", \"value\": \"$${v}\"}}]"; \
    done
endif

install-operator:
	@echo "####### Installing operator..."
ifndef NAMESPACE
	@echo "WARN: no namespace specified, using default namespace $(DEFAULT_NS)"
	$(eval NAMESPACE=$(DEFAULT_NS))
endif
	cd $(KUST_TMP)/$(KUSTOMIZE_DIR) && kustomize edit set namespace $(NAMESPACE)
	cd $(KUST_TMP)/install/overlays/platform && kustomize edit set namespace $(NAMESPACE)
	kubectl apply -k $(KUST_TMP)/$(KUSTOMIZE_DIR) --server-side --force-conflicts

install-registry:
	@echo "####### Installing container registry..."
ifdef REGISTRY
	@echo "INFO: Setting IntegrationPlatform container registry to $(REGISTRY)"
	@sed -i 's/address: .*/address: $(REGISTRY)/' $(KUST_TMP)/install/overlays/platform/integration-platform.yaml
	kubectl apply -k $(KUST_TMP)/install/overlays/platform --server-side -n $(NAMESPACE) --force-conflicts
else
	$(eval MINIKUBE_REGISTRY=$(shell kubectl -n kube-system get service registry -o jsonpath='{.spec.clusterIP}' 2> /dev/null))
ifneq ($(MINIKUBE_REGISTRY),"")
	@echo "INFO: Looks like you're on Minikube. Setting IntegrationPlatform container registry to $(MINIKUBE_REGISTRY)"
	@sed -i 's/address: .*/address: $(MINIKUBE_REGISTRY)/' $(KUST_TMP)/install/overlays/platform/integration-platform.yaml
	kubectl apply -k $(KUST_TMP)/install/overlays/platform --server-side -n $(NAMESPACE) --force-conflicts
endif
endif

install-k8s-global: DEFAULT_NS="camel-k"
install-k8s-global: KUSTOMIZE_DIR="install/overlays/kubernetes/descoped"
install-k8s-global: clone-kustomize-dir set-operator-id set-operator-env install-operator install-registry

install-k8s-ns: DEFAULT_NS="default"
install-k8s-ns: KUSTOMIZE_DIR="install/overlays/kubernetes/namespaced"
install-k8s-ns: clone-kustomize-dir set-operator-id set-operator-env install-operator install-registry

install-openshift-global: DEFAULT_NS="camel-k"
install-openshift-global: KUSTOMIZE_DIR="install/overlays/openshift/descoped"
install-openshift-global: clone-kustomize-dir set-operator-id set-operator-env install-operator

install-openshift-ns: DEFAULT_NS="default"
install-openshift-ns: KUSTOMIZE_DIR="install/overlays/openshift/namespaced"
install-openshift-ns: clone-kustomize-dir set-operator-id set-operator-env install-operator

uninstall:
ifdef NAMESPACE
	kubectl delete deploy -l app=camel-k -n $(NAMESPACE)
	kubectl delete configmap,secret,sa,rolebindings,clusterrolebindings,roles,clusterroles,integrationplatform -l app=camel-k -n $(NAMESPACE)
else
	kubectl delete deploy -l app=camel-k
	kubectl delete configmap,secret,sa,rolebindings,clusterrolebindings,roles,clusterroles,integrationplatform -l app=camel-k
endif

install-crds:
	kubectl apply -k install/base/config/crd --server-side --force-conflicts

uninstall-crds:
	kubectl delete crd -l app=camel-k

uninstall-all: uninstall uninstall-crds

#####
# END Local installation procedure. Handy for development purpose
#####

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

VERSIONFILE := pkg/util/defaults/defaults.go
VERSION ?= 2.2.0-SNAPSHOT
LAST_RELEASED_IMAGE_NAME := camel-k-operator
LAST_RELEASED_VERSION ?= 2.1.0
DEFAULT_RUNTIME_VERSION := 3.2.3
BUILDAH_VERSION := 1.30.0
KANIKO_VERSION := 1.9.1
CONTROLLER_GEN_VERSION := v0.6.1
CODEGEN_VERSION := v0.27.4
OPERATOR_SDK_VERSION := v1.28.0
KUSTOMIZE_VERSION := v4.5.4
OPM_VERSION := v1.24.0
BASE_IMAGE := eclipse-temurin:17
LOCAL_REPOSITORY := /etc/maven/m2
IMAGE_NAME ?= docker.io/apache/camel-k
# Default value, change to your architecture accordingly
IMAGE_ARCH ?= amd64
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
LINT_GOGC := 10
LINT_DEADLINE := 10m
DEBUG_MODE ?= false

# olm bundle vars
MANAGER := config/manager
MANIFESTS := config/manifests
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
STAGING_IMAGE_NAME := docker.io/camelk/camel-k
STAGING_RUNTIME_REPO :=

# Kamelets options
INSTALL_DEFAULT_KAMELETS ?= true
KAMELET_CATALOG_REPO := https://github.com/apache/camel-kamelets.git
# Make sure to use a released tag or empty if you want to get the latest development bits
KAMELET_CATALOG_REPO_TAG := v4.0.1

# When performing integration tests, it is not necessary to always execute build, especially
# in e2e tests when lots of tests are being executed sequentially & the build has already taken place.
# In this case, if DO_TEST_PREBUILD is set to false then TEST_PREBUILD will be set to blank
# and those rules with a dependency on the build rule will no longer call it (see do-build)
DO_TEST_PREBUILD ?= true
TEST_PREBUILD = build

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
	@sed -i 's/^STAGING_RUNTIME_REPO :=.*$//STAGING_RUNTIME_REPO :=/' ./script/Makefile

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
	@echo "  // BuildahVersion -- " >> $(VERSIONFILE)
	@echo "  BuildahVersion = \"$(BUILDAH_VERSION)\"" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)
	@echo "  // KanikoVersion -- " >> $(VERSIONFILE)
	@echo "  KanikoVersion = \"$(KANIKO_VERSION)\"" >> $(VERSIONFILE)
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

ci-build: clean codegen set-version check-licenses dir-licenses build-kamel cross-compile

do-build: gotestfmt-install
ifeq ($(DO_TEST_PREBUILD),true)
TEST_PREBUILD = build
else
TEST_PREBUILD =
endif

gotestfmt-install:
ifeq (, $(shell command -v gotestfmt 2> /dev/null))
	go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
endif

test: do-build
	@echo "####### Running unit test..."
	go test ./...

test-fmt: do-build
	@echo "####### Running unit test..."
	go test -v ./... -json 2>&1 | gotestfmt

#
# Common tests that do not require any customized operator setting. They can leverage a unique namespaced operator installation to reduce
# the time to complete (they are used also as smoke test for nightly release)
#
test-common: do-build
	FAILED=0; STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 10m -v ./e2e/common/support/startup_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v ./e2e/common/languages -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v ./e2e/common/cli -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v ./e2e/common/config -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v ./e2e/common/misc -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 60m -v ./e2e/common/traits -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 20m -v ./e2e/common/runtimes -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 10m -v ./e2e/common/support/teardown_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	exit $${FAILED}

#
# Smoke tests are a subset of common tests used to quickly verify the operator
#
test-smoke: do-build
	FAILED=0; STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 10m -v ./e2e/common/support/startup_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v ./e2e/common/languages -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v \
		./e2e/common/misc/default.go \
		./e2e/common/misc/cron_test.go \
		./e2e/common/misc/kamelet_test.go \
		./e2e/common/misc/pipe_test.go \
	 	-tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 30m -v \
		./e2e/common/traits/default.go \
		./e2e/common/traits/camel_test.go \
		./e2e/common/traits/container_test.go \
		./e2e/common/traits/openapi_test.go \
		./e2e/common/traits/service_test.go \
	 	-tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 10m -v ./e2e/common/support/teardown_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	exit $${FAILED}

#
# Common tests that require some particular operator setting or need to be installed in multiple namespaces
#
test-common-with-custom-install: do-build
	FAILED=0; STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 90m -v ./e2e/commonwithcustominstall -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT) || FAILED=1; \
	exit $${FAILED}

#
# Installation procedure test using CLi and Kustomize methodologies
#
test-install: do-build
	FAILED=0; STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 40m -v ./e2e/install/cli -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 40m -v ./e2e/install/kustomize -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 40m -v ./e2e/install/helm -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT) || FAILED=1; \
	exit $${FAILED}

#
# Installation procedure test using OLM (requires cluster to be setup accordingly)
#
test-install-olm: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 30m -v ./e2e/install/olm -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT)

#
# Upgrade procedure test (requires cluster to be setup accordingly, ie OLM)
#
test-install-upgrade: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 30m -v ./e2e/install/upgrade -tags=integration $(TEST_INSTALL_RUN) $(GOTESTFMT)

#
# Knative tests that require the presence of KNative configuration
#
test-knative: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 10m -v ./e2e/knative/support/startup_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 60m -v ./e2e/knative -tags=integration $(TEST_KNATIVE_RUN) $(GOTESTFMT) || FAILED=1; \
	go test -timeout 10m -v ./e2e/knative/support/teardown_test.go -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN) $(GOTESTFMT) || FAILED=1; \
	exit $${FAILED}

#
# Builder tests that require a custom build strategy
#
test-builder: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)"; \
	go test -timeout 30m -v ./e2e/builder -tags=integration $(TEST_BUILDER_RUN) $(GOTESTFMT)

#
# Telemetry tests that require the configuration of telemtry endpoints
#
test-telemetry: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 30m -v ./e2e/telemetry -tags=integration $(TEST_TELEMETRY_RUN) $(GOTESTFMT)

#
# Quarkus native test which require a lower memory resources
#
test-quarkus-native: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 180m -v ./e2e/native -tags=integration $(TEST_QUARKUS_RUN) $(GOTESTFMT)

#
# Quarkus native test which require a higher memory resources
#
test-quarkus-native-high-memory: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 180m -v ./e2e/native -tags=integration,high_memory $(TEST_QUARKUS_RUN) $(GOTESTFMT)

build-kamel:
	@echo "####### Building kamel CLI for $(GOOS) $(IMAGE_ARCH) architecture..."
	CGO_ENABLED=0 GOARCH=$(IMAGE_ARCH) go build $(GOFLAGS) -o build/_output/bin/kamel-$(IMAGE_ARCH) ./cmd/kamel/*.go
	# Symbolic link to a local CLI
	ln -sf build/_output/bin/kamel-$(IMAGE_ARCH) ./kamel

build-resources:
	./script/get_catalog.sh $(DEFAULT_RUNTIME_VERSION) $(STAGING_RUNTIME_REPO)
	go generate ./pkg/...

bundle-kamelets:
	@echo "Preparing Kamelets bundle resource..."
ifneq (,$(findstring release,$(MAKECMDGOALS)))
ifneq (,$(findstring $(KAMELET_CATALOG_REPO_TAG), main))
	@echo "You cannot set KAMELET_CATALOG_REPO_TAG=$(KAMELET_CATALOG_REPO_TAG) when doing a release"
	@exit 1
endif
endif
	./script/bundle_kamelets.sh $(KAMELET_CATALOG_REPO) $(KAMELET_CATALOG_REPO_TAG)

build-compile-integration-tests:
	@echo "####### Compiling integration tests..."
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

lint:
	GOGC=$(LINT_GOGC) golangci-lint run --config .golangci.yml --out-format tab --deadline $(LINT_DEADLINE) --verbose

lint-fix:
	GOGC=$(LINT_GOGC) golangci-lint run --config .golangci.yml --out-format tab --deadline $(LINT_DEADLINE) --fix

dir-licenses:
	./script/vendor-license-directory.sh

check-licenses:
	./script/check_licenses.sh

# The below are dependencies needed for maven structured logs. We must bundle into the final container image.
maven-overlay:
	@echo "####### Preparing maven dependencies bundle..."
	mkdir -p build/_maven_overlay
	./script/maven_overlay.sh build/_maven_overlay

kamel-overlay:
	mkdir -p build/_output/bin
ifeq ($(shell uname -s 2>/dev/null || echo Unknown),Linux)
	@echo "####### Copying Linux platform arch $(IMAGE_ARCH) CLI into output build directory..."
else
	@echo "####### (Re)Building a Linux platform arch $(IMAGE_ARCH) CLI "
	@echo "####### Likely you're on a non Linux host, so I need to build a Linux CLI to bundle in the container image"
	GOOS=linux GOARCH=$(IMAGE_ARCH) go build $(GOFLAGS) -o build/_output/bin/kamel-$(IMAGE_ARCH) ./cmd/kamel/*.go
endif
	cp build/_output/bin/kamel-$(IMAGE_ARCH) build/_output/bin/kamel

TARGET_STAGE := base
ifeq ($(DEBUG_MODE),true)
	TARGET_STAGE := debug
	CUSTOM_IMAGE := $(CUSTOM_IMAGE)-debug
endif

DOCKER_TAG := $(CUSTOM_IMAGE):$(CUSTOM_VERSION)
ifneq ($(IMAGE_ARCH), amd64)
	DOCKER_TAG := $(DOCKER_TAG)-$(IMAGE_ARCH)
endif

images: build kamel-overlay maven-overlay bundle-kamelets
ifneq (,$(findstring SNAPSHOT,$(DEFAULT_RUNTIME_VERSION)))
	./script/package_maven_artifacts.sh -s "$(STAGING_RUNTIME_REPO)" -d "$(CAMEL_K_RUNTIME_DIR)" $(DEFAULT_RUNTIME_VERSION)
endif
	@echo "####### Building Camel K operator arch $(IMAGE_ARCH) container image..."
	mkdir -p build/_maven_output
	docker build --target $(TARGET_STAGE) --platform=linux/$(IMAGE_ARCH) -t $(DOCKER_TAG) -f build/Dockerfile .

# Mainly used for internal CI purposes
image-push:
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)

images-all: images
	make IMAGE_ARCH=arm64 images

images-push:
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64

images-push-staging:
	docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION) $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)
	docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64 $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)-arm64
	docker push $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)
	docker push $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)-arm64
	# TODO: we can evaluate the usage of manifest and try the following
	# docker manifest create $(CUSTOM_IMAGE):$(CUSTOM_VERSION) \
	# --amend $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-amd64 \
	# --amend $(CUSTOM_IMAGE):$(CUSTOM_VERSION)-arm64
	# docker manifest push --purge $(CUSTOM_IMAGE):$(CUSTOM_VERSION)

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

release-staging: clean codegen set-version build-resources check-licenses images-all images-push-staging cross-compile git-tag

# Build images
images-nightly: clean codegen set-version build-resources images

# Release artifacts
release-nightly: cross-compile images-push

release-helm:
	./script/release_helm.sh ${VERSION}

release-kustomize:
	RELEASE_VERSION=$(CUSTOM_VERSION) \
	RELEASE_NAME=$(PACKAGE) \
	./script/release_kustomize.sh

get-staging-repo:
	@echo $(or ${STAGING_RUNTIME_REPO},https://repository.apache.org/content/repositories/snapshots@id=apache-snapshots@snapshots)

.PHONY: do-build build build-kamel build-resources dep codegen images images-push images-push-staging test check clean release cross-compile package-examples set-version git-tag check-licenses build-resources release-helm release-staging release-nightly get-staging-repo get-version bundle-kamelets
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
	$(OPERATOR_SDK) generate kustomize manifests --apis-dir $(addprefix api_, $@) -q
	@# Adds the licence header to the csv file.
	./script/add_license.sh config/manifests/bases ./script/headers/yaml.txt
	./script/add_createdAt.sh config/manifests/bases
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
	@if grep -q replaces $(CSV_PATH); \
		then sed -i 's/^  replaces: .*/  replaces: $(CSV_REPLACES)/' $(CSV_PATH); \
		else sed -i '/  version: ${CSV_VERSION}/a \ \ replaces: $(CSV_REPLACES)' $(CSV_PATH); \
	fi

bundle: set-version kustomize operator-sdk pre-bundle $(BUNDLE_CAMEL_APIS)
	@# Display BUNDLE_METADATA_OPTS for debugging
	$(info BUNDLE_METADATA_OPTS=$(BUNDLE_METADATA_OPTS))
	@# Sets the operator image to the preferred image:tag
	@cd config/manifests && $(KUSTOMIZE) edit set image $(IMAGE_NAME)=$(CUSTOM_IMAGE):$(CUSTOM_VERSION)
	@# Build kustomize manifests
	@$(KUSTOMIZE) build config/manifests | \
		$(OPERATOR_SDK) generate bundle \
			-q --overwrite --version $(OPERATOR_VERSION) \
			--kustomize-dir config/manifests $(BUNDLE_METADATA_OPTS)
	@# Move the dockerfile into the bundle directory
	@mv bundle.Dockerfile bundle/Dockerfile && sed -i 's/bundle\///g' bundle/Dockerfile
	@# Adds the licence headers to the csv file
	./script/add_license.sh bundle/manifests ./script/headers/yaml.txt
# Rename the CSV name to conform with the existing released operator versions
# This cannot happen in pre-bundle as the operator-sdk generation expects a CSV name the same as PACKAGE
	@sed -i "s/^  name: $(CSV_NAME)/  name: $(CSV_PRODUCTION_NAME)/" $(CSV_PRODUCTION_PATH)
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build: bundle
	cd bundle && docker build -f Dockerfile -t $(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION) .

.PHONY: bundle-push bundle-index

bundle-push: bundle-build
	docker push $(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION)

#
# Builds a test catalog index for installing the operator via an OLM
#
bundle-index: opm yq
	BUNDLE_INDEX=$(BUNDLE_INDEX) INDEX_DIR=$(INDEX_DIR) PACKAGE=$(PACKAGE) YQ=$(YQ) \
	OPM=$(OPM) BUNDLE_IMAGE=$(BUNDLE_IMAGE_NAME):$(CUSTOM_VERSION) CSV_NAME=$(CSV_PRODUCTION_NAME) \
	CSV_SKIPS=$(CSV_SKIP_RANGE) CSV_REPLACES=$(CSV_REPLACES) CHANNELS="$(CHANNELS)" \
	./script/build_bundle_index.sh

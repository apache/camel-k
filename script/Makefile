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

VERSIONFILE := pkg/util/defaults/defaults.go
VERSION ?= 1.9.3-SNAPSHOT
OPERATOR_VERSION := $(subst -SNAPSHOT,,$(VERSION))
LAST_RELEASED_IMAGE_NAME := camel-k-operator
LAST_RELEASED_VERSION := 1.9.2
RUNTIME_VERSION := 1.13.0
BUILDAH_VERSION := 1.14.0
KANIKO_VERSION := 0.17.1
INSTALL_DEFAULT_KAMELETS := true
CONTROLLER_GEN_VERSION := v0.4.1
OPERATOR_SDK_VERSION := v1.14.0
KUSTOMIZE_VERSION := v4.1.2
OPM_VERSION := v1.21.0
BASE_IMAGE := adoptopenjdk/openjdk11:slim
LOCAL_REPOSITORY := /tmp/artifacts/m2
IMAGE_NAME := docker.io/apache/camel-k

#
# Situations when user wants to override
# the image name and version
# - used in kustomize install
# - used in making bundle
# - need to preserve original image and version as used in other files
#
CUSTOM_IMAGE ?= $(IMAGE_NAME)
CUSTOM_VERSION ?= $(VERSION)

METADATA_IMAGE_NAME := $(CUSTOM_IMAGE)-metadata
BUNDLE_IMAGE_NAME ?= $(CUSTOM_IMAGE)-bundle
RELEASE_GIT_REMOTE := origin
GIT_COMMIT := $(shell git rev-list -1 HEAD)
LINT_GOGC := 10
LINT_DEADLINE := 10m


# olm bundle vars
CHANNELS ?= $(OPERATOR_VERSION),candidate,stable,latest
DEFAULT_CHANNEL ?= $(OPERATOR_VERSION)
PACKAGE := camel-k
CSV_VERSION := $(CUSTOM_VERSION:-SNAPSHOT=)
CSV_NAME := $(PACKAGE).v$(CSV_VERSION)
CSV_DISPLAY_NAME := Camel K Operator
CSV_SUPPORT := Camel
CSV_REPLACES := $(LAST_RELEASED_IMAGE_NAME).v$(LAST_RELEASED_VERSION)
CSV_FILENAME := $(PACKAGE).clusterserviceversion.yaml
CSV_PATH := config/manifests/bases/$(CSV_FILENAME)

# Used to push pre-release artifacts
STAGING_IMAGE_NAME := docker.io/camelk/camel-k

STAGING_RUNTIME_REPO :=

# Define here the repo containing the default Kamelet catalog (if any)
KAMELET_CATALOG_REPO := https://github.com/apache/camel-kamelets.git
# Optional branch for the default Kamelet catalog (change this to a tag before release)
KAMELET_CATALOG_REPO_BRANCH := v0.8.1

# When performing integration tests, it is not necessary to always execute build, especially
# in e2e tests when lots of tests are being executed sequentially & the build has already taken place.
# In this case, if DO_TEST_PREBUILD is set to false then TEST_PREBUILD will be set to blank
# and those rules with a dependency on the build rule will no longer call it (see do-build)
DO_TEST_PREBUILD ?= true
TEST_PREBUILD = build

# OLM (Operator Lifecycle Manager and Operator Hub): uncomment to override operator settings at build time
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultOperatorName=camel-k-operator
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultPackage=camel-k
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultChannel=alpha
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultSource=community-operators
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultSourceNamespace=openshift-marketplace
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultStartingCSV=
#GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/olm.DefaultGlobalNamespace=openshift-operators

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Build
ifdef GIT_COMMIT
GOLDFLAGS += -X github.com/apache/camel-k/pkg/util/defaults.GitCommit=$(GIT_COMMIT)
else
$(warning Could not retrieve a valid Git Commit)
endif

GOFLAGS = -ldflags "$(GOLDFLAGS)" -trimpath

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
default: test

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
	@echo "  DefaultRuntimeVersion = \"$(RUNTIME_VERSION)\"" >> $(VERSIONFILE)
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
	@echo "//GitCommit must be provided during application build" >> $(VERSIONFILE)
	@echo "var GitCommit string" >> $(VERSIONFILE)
	@echo "" >> $(VERSIONFILE)

	gofmt -w pkg/util/defaults/defaults.go

generate: generate-deepcopy generate-crd generate-client generate-doc generate-keda generate-strimzi

generate-client:
	./script/gen_client.sh

generate-crd: controller-gen
	CONTROLLER_GEN=$(CONTROLLER_GEN) ./script/gen_crd.sh

generate-doc:
	./script/gen_doc.sh

generate-deepcopy: controller-gen
	cd pkg/apis/camel && $(CONTROLLER_GEN) paths="./..." object

generate-keda:
	cd addons/keda/duck && $(CONTROLLER_GEN) paths="./..." object

generate-strimzi:
	cd addons/strimzi/duck && $(CONTROLLER_GEN) paths="./..." object
	./script/gen_client_strimzi.sh

build: build-resources build-kamel build-compile-integration-tests build-submodules

do-build:
ifeq ($(DO_TEST_PREBUILD),true)
TEST_PREBUILD = build
else
TEST_PREBUILD =
endif

test: do-build
	go test ./...

#
# Note
# This should execute all tests and not exit if one of the suites fail
#
test-integration: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/common -tags=integration $(TEST_INTEGRATION_COMMON_RUN); \
	go test -timeout 60m -v ./e2e/common/build -tags=integration $(TEST_INTEGRATION_COMMON_BUILD_RUN); \
	go test -timeout 60m -v ./e2e/common/cli -tags=integration $(TEST_INTEGRATION_COMMON_CLI_RUN); \
	go test -timeout 60m -v ./e2e/common/config -tags=integration $(TEST_INTEGRATION_COMMON_CONFIG_RUN); \
	go test -timeout 60m -v ./e2e/common/languages -tags=integration $(TEST_INTEGRATION_COMMON_LANG_RUN); \
	go test -timeout 60m -v ./e2e/common/traits -tags=integration $(TEST_INTEGRATION_COMMON_TRAITS_RUN)

test-knative: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/knative -tags=integration $(TEST_KNATIVE_RUN)

test-builder: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/builder -tags=integration $(TEST_BUILDER_RUN)

test-local: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 30m -v ./e2e/local -tags=integration $(TEST_LOCAL_RUN)

test-kamel-cli: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/common/cli -tags=integration $(TEST_KAMEL_CLI)

test-kustomize: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/common/kustomize -tags=integration $(TEST_KUSTOMIZE_RUN)

test-quarkus-native: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/native -tags=integration $(TEST_QUARKUS_RUN)

test-service-binding: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/service-binding -tags=integration $(TEST_SERVICE_RUN)

test-upgrade: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/upgrade -tags=integration $(TEST_UPGRADE_RUN)

test-registry-maven-wagon: do-build
	STAGING_RUNTIME_REPO="$(STAGING_RUNTIME_REPO)" \
	go test -timeout 60m -v ./e2e/registry -tags=integration $(TEST_REGISTRY_MAVEN_WAGON_RUN)

build-kamel:
# Ensure the binary is statically linked when building on Linux due to ABI changes in newer glibc 2.32, otherwise
# it would not run on older versions. See https://github.com/apache/camel-k/pull/2141#issuecomment-800990117
ifeq ($(shell uname -s 2>/dev/null || echo Unknown),Linux)
	CGO_ENABLED=0 go build $(GOFLAGS) -o kamel ./cmd/kamel/*.go
else
	go build $(GOFLAGS) -o kamel ./cmd/kamel/*.go
endif

build-resources:
	./script/get_catalog.sh $(RUNTIME_VERSION) $(STAGING_RUNTIME_REPO)
	go generate ./pkg/...

bundle-kamelets:
ifneq (,$(findstring release,$(MAKECMDGOALS)))
ifneq (,$(findstring $(KAMELET_CATALOG_REPO_BRANCH), main))
	@echo "You cannot set KAMELET_CATALOG_REPO_BRANCH=$(KAMELET_CATALOG_REPO_BRANCH) when doing a release"
	@exit 1
endif
endif
	./script/bundle_kamelets.sh $(KAMELET_CATALOG_REPO) $(KAMELET_CATALOG_REPO_BRANCH)

build-submodules:
	./script/build_submodules.sh

build-compile-integration-tests:
	go test -c -tags="integration" ./e2e/common
	go test -c -tags="integration" ./e2e/knative
	go test -c -tags="integration" ./e2e/builder

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

dep:
	go mod tidy

lint:
	GOGC=$(LINT_GOGC) golangci-lint run --out-format tab --deadline $(LINT_DEADLINE)

lint-fix:
	GOGC=$(LINT_GOGC) golangci-lint run --out-format tab --deadline $(LINT_DEADLINE) --fix


check-licenses:
	./script/check_licenses.sh

check-platform:
	./script/check_platform.sh

# The below are dependencies needed for maven structured logs. We must bundle into the final container image.
maven-overlay:
	mkdir -p build/_maven_overlay
	./script/maven_overlay.sh build/_maven_overlay

images: bundle-kamelets test maven-overlay
	mkdir -p build/_maven_output
	mkdir -p build/_output/bin
ifneq ($(shell uname -s 2>/dev/null || echo Unknown),Linux)
	GOOS=linux go build $(GOFLAGS) -o build/_output/bin/kamel ./cmd/kamel/*.go
else
	cp kamel build/_output/bin
endif
	docker build -t $(CUSTOM_IMAGE):$(CUSTOM_VERSION) -f build/Dockerfile .

images-dev: bundle-kamelets test package-artifacts maven-overlay
	mkdir -p build/_maven_output
	mkdir -p build/_output/bin
ifneq ($(shell uname -s 2>/dev/null || echo Unknown),Linux)
	GOOS=linux go build $(GOFLAGS) -o build/_output/bin/kamel ./cmd/kamel/*.go
else
	cp kamel build/_output/bin
endif
	docker build -t $(CUSTOM_IMAGE):$(CUSTOM_VERSION) -f build/Dockerfile .

images-push:
	docker push $(CUSTOM_IMAGE):$(CUSTOM_VERSION)

images-push-staging:
	docker tag $(CUSTOM_IMAGE):$(CUSTOM_VERSION) $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)
	docker push $(STAGING_IMAGE_NAME):$(CUSTOM_VERSION)

get-image:
	@echo $(CUSTOM_IMAGE)

get-version:
	@echo $(CUSTOM_VERSION)

get-last-released-img-name:
	@echo $(LAST_RELEASED_IMAGE_NAME)

get-last-released-version:
	@echo $(LAST_RELEASED_VERSION)

get-csv-name:
	@echo $(CSV_NAME)

set-version:
	./script/set_version.sh $(CUSTOM_VERSION) $(CUSTOM_IMAGE)

set-module-version:
	./script/set_go_modules_version.sh $(CUSTOM_VERSION)

git-tag:
	./script/git_tag.sh $(CUSTOM_VERSION) $(RELEASE_GIT_REMOTE)

cross-compile:
	# we must wrap the goldflags parameters with quotes as they will need to
	# be processed as a single argument by the cross compile script
	./script/cross_compile.sh $(CUSTOM_VERSION) $(subst ","\",$(GOFLAGS))

package-examples:
	./script/package_examples.sh $(CUSTOM_VERSION)

package-artifacts:
	# useful for development purpose only in order to include any local development runtime dependency
	./script/package_maven_artifacts.sh $(RUNTIME_VERSION) $(CAMEL_K_RUNTIME_DIR)

release: clean codegen set-module-version set-version build-resources check-licenses build images images-push cross-compile package-examples git-tag

release-staging: clean codegen set-module-version set-version build-resources check-licenses build images images-push-staging cross-compile package-examples git-tag

release-nightly: clean codegen set-module-version set-version build-resources build images images-push cross-compile package-examples

release-helm:
	./script/release_helm.sh

install-crc:
	./script/install_crc.sh $(CUSTOM_VERSION)
install-minikube:
	./script/install_minikube.sh

get-staging-repo:
	@echo $(or ${STAGING_RUNTIME_REPO},https://repository.apache.org/content/repositories/snapshots@id=apache-snapshots@snapshots)

.PHONY: do-build build build-kamel build-resources dep codegen images images-dev images-push images-push-staging test check test-integration clean release cross-compile package-examples set-version git-tag release-notes check-licenses generate-deepcopy generate-client generate-doc build-resources release-helm release-staging release-nightly get-staging-repo get-version build-submodules set-module-version bundle-kamelets generate-keda generate-strimzi
.PHONY: controller-gen kubectl kustomize operator-sdk

# find or download controller-gen if necessary
controller-gen:
ifeq (, $(shell command -v controller-gen 2> /dev/null))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell command -v controller-gen 2> /dev/null)
endif

.PHONY: kubectl kustomize operator-sdk opm

kubectl:
ifeq (, $(shell command -v kubectl 2> /dev/null))
	$(error "No kubectl found in PATH. Please install and re-run")
endif

kustomize:
ifeq (, $(shell command -v kustomize 2> /dev/null))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION) ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell command -v kustomize 2> /dev/null)
endif

operator-sdk:
ifeq (, $(shell command -v operator-sdk 2> /dev/null))
	@{ \
	set -e ;\
	if [ "$(shell uname -s 2>/dev/null || echo Unknown)" == "Darwin" ] ; then \
		curl \
			-L https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_darwin_amd64 \
			-o operator-sdk ; \
	else \
		curl \
			-L https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_linux_amd64 \
			-o operator-sdk ; \
	fi ;\
	chmod +x operator-sdk ;\
	mv operator-sdk $(GOBIN)/ ;\
	}
OPERATOR_SDK=$(GOBIN)/operator-sdk
else
	@{ \
	echo -n "operator-sdk already installed: "; \
  operator-sdk version | sed -n 's/.*"v\([^"]*\)".*/\1/p'; \
	echo " If this is less than $(OPERATOR_SDK_VERSION) then please consider moving it aside and allowing the approved version to be downloaded."; \
	}
OPERATOR_SDK=$(shell command -v operator-sdk 2> /dev/null)
endif

opm:
ifeq (, $(shell command -v opm 2> /dev/null))
	@OPM_VERSION=$(OPM_VERSION) ./script/build_install_opm.sh
OPM=$(GOBIN)/opm
else
OPM=$(shell command -v opm 2> /dev/null)
endif

.PHONY: generate-crd $(BUNDLE_CAMEL_APIS) bundle bundle-build

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
	@sed -i 's/^  replaces: .*/  replaces: $(CSV_REPLACES)/' $(CSV_PATH)
	@sed -i 's/^  version: .*/  version: $(CSV_VERSION)/' $(CSV_PATH)
	@sed -i '/  version: ${CSV_VERSION}/a \ \ replaces: $(CSV_REPLACES)' $(CSV_PATH)

bundle: set-version generate-crd $(BUNDLE_CAMEL_APIS) kustomize operator-sdk pre-bundle
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
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build: bundle
	cd bundle && docker build -f Dockerfile -t $(BUNDLE_IMAGE_NAME) .

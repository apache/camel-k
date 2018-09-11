VERSION := $(shell ./build/get_version.sh)

build: build-runtime build-embed-resources build-operator build-kamel

build-operator:
	go build -o camel-k-operator ./cmd/camel-k-operator/*.go

build-kamel:
	go build -o kamel ./cmd/kamel/*.go

build-embed-resources:
	./build/embed_resources.sh deploy

build-runtime:
	mvn clean install -f ./runtime/pom.xml

dep:
	dep ensure -v

clean:
	mvn clean -f ./runtime/pom.xml
	go clean
	rm -f camel-k-operator
	rm -f kamel

codegen:
	./tmp/codegen/update-generated.sh

images: images-build

images-build:
	operator-sdk build docker.io/apache/camel-k:$(VERSION)

images-push:
	docker push docker.io/apache/camel-k:$(VERSION)

test: check
check:
	go test ./...

test-integration: check-integration
check-integration: require-kubeconfig require-namespace
	go test ./... -tags=integration

ifndef WATCH_NAMESPACE
require-namespace:
	$(error WATCH_NAMESPACE not set)
else
require-namespace:
	@echo "WATCH_NAMESPACE set"
endif
ifndef KUBERNETES_CONFIG
require-kubeconfig:
	$(error KUBERNETES_CONFIG not set)
else
require-kubeconfig:
	@echo "KUBERNETES_CONFIG set"
endif

.PHONY: build build-operator build-kamel build-embed-resources build-runtime dep codegen images images-build images-push test check test-integration check-integration clean 
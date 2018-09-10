VERSION := $(shell ./build/get_version.sh)

build: build-runtime build-embed-resources get-go-deps build-operator build-kamel

get-go-deps:
	go get -v -d ./...

build-operator:
	go build -o camel-k-operator ./cmd/camel-k-operator/*.go

build-kamel:
	go build -o kamel ./cmd/kamel/*.go

build-embed-resources:
	./build/embed_resources.sh deploy

build-runtime:
	mvn clean install -f ./runtime/pom.xml

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
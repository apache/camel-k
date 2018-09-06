VERSION := $(shell ./build/get_version.sh)

install:
	go build -o camel-k ./cmd/camel-k/*.go

images:
	operator-sdk build docker.io/apache/camel-k:$(VERSION)

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
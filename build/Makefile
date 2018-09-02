VERSION := $(shell ./build/get_version.sh)

compile:
	go build -o camel-k ./cmd/camel-k/*.go

images:
	operator-sdk build docker.io/apache/camel-k:$(VERSION)
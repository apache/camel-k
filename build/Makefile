build: build-runtime build-operator build-kamel test

build-operator: build-embed-resources
	go build -o camel-k-operator ./cmd/camel-k-operator/*.go

build-kamel:
	go build -o kamel ./cmd/kamel/*.go

build-embed-resources:
	./build/embed_resources.sh deploy

build-runtime:
	mvn clean install -f ./runtime/pom.xml

release: prepare-release build images-build images-push test-integration cross-compile git-tag

prepare-release:
	./build/prepare_release.sh

new-version:
	./build/next_snapshot.sh

cross-compile:
	./build/cross_compile.sh

git-tag:
	./build/git_tag.sh

dep:
	dep ensure -v

generate:
	operator-sdk generate k8s

clean:
	mvn clean -f ./runtime/pom.xml
	go clean
	rm -f camel-k-operator
	rm -f kamel
	rm -rf tmp/_maven_output

codegen:
	./tmp/codegen/update-generated.sh

images: images-build

images-build:
	./build/images_build.sh

images-push:
	./build/images_push.sh

test: check
check:
	go test ./...

test-integration: check-integration
check-integration:
	go test ./... -tags=integration



.PHONY: build build-operator build-kamel build-embed-resources build-runtime dep codegen images images-build images-push test check test-integration check-integration clean release prepare-release cross-compile new-version git-tag

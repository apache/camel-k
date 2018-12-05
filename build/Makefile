build: build-runtime build-operator build-kamel build-compile-integration-tests test

build-go: build-embed-resources build-operator build-kamel

build-operator: build-embed-resources
	go build -o camel-k ./cmd/camel-k/*.go

build-kamel:
	go build -o kamel ./cmd/kamel/*.go

build-embed-resources:
	./build/embed_resources.sh deploy

build-compile-integration-tests:
	go test -c -tags=integration ./test/*.go

build-runtime:
	./mvnw clean install -f ./runtime/pom.xml

release: clean prepare-release build images-build images-push cross-compile git-tag

prepare-release:
	./build/prepare_release.sh

new-version: increment-snapshot build images-build images-push

increment-snapshot:
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
	./mvnw clean -f ./runtime/pom.xml
	go clean
	rm -f camel-k
	rm -f kamel
	rm -rf tmp/_maven_output

codegen:
	./tmp/codegen/update-generated.sh

images: images-build

images-build:
	./build/images_build.sh

images-push:
	./build/images_push.sh

install: install-minishift
install-minishift:
	./build/install_minishift.sh

install-minikube:
	./build/install_minikube.sh

test: check
check:
	go test ./...

test-integration: check-integration
check-integration:
	go test ./... -tags=integration

lint:
	golangci-lint run

.PHONY: build build-operator build-kamel build-embed-resources build-runtime dep codegen images images-build images-push test check test-integration check-integration clean release prepare-release cross-compile new-version git-tag increment-snapshot install-minishift

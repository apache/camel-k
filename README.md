# Apache Camel K

Apache Camel K (a.k.a. Kamel) is a lightweight integration framework built from Apache Camel that runs natively on Kubernetes and is specifically designed for serverless and microservice architectures.

## Build

In order to build the project follow these steps:
- this project is supposed to be cloned in `$GOPATH/src/github.com/apache/camel-k`
- install dep: https://github.com/golang/dep
- install operator-sdk: https://github.com/operator-framework/operator-sdk
- dep ensure -v
- make build

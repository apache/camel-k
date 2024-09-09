# Camel K End-To-End tests

This directory contains the suite of test that are run on a CI to ensure the stability of the product and no regression are introduced at each PR. The full documentation can be found at https://camel.apache.org/camel-k/next/contributing/e2e.html

## Environment variables

You can set some environment variables to change the behavior of the E2E test suite.

| Env                                     | Default                                 | Description                                                                                                                                   |
|-----------------------------------------|-----------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| CAMEL_K_TEST_SAVE_FAILED_TEST_NAMESPACE | false                                   | Used to not remove the temporary test namespaces after the test run. Enables better analysis of resources after the test                      |
| CAMEL_K_TEST_LOG_LEVEL                  | info                                    | Logging level used to run the tests and used in Maven commands run by the operator (if level is `debug` the Maven commands use `-X` option).  |
| CAMEL_K_TEST_MAVEN_CLI_OPTIONS          | {}                                      | Maven CLI options used to run Camel K integrations during the tests.                                                                          |
| CAMEL_K_TEST_OPERATOR_IMAGE             | docker.io/apache/camel-k:2.4.0-SNAPSHOT | Camel K operator image used in operator installation.                                                                                         |
| CAMEL_K_TEST_OPERATOR_IMAGE_PULL_POLICY | -                                       | Operator image pull policy.                                                                                                                   |
| CAMEL_K_TEST_IMAGE_NAME                 | docker.io/apache/camel-k                | Camel K operator image name used in operator installation.                                                                                    |
| CAMEL_K_TEST_IMAGE_VERSION              | 2.4.0-SNAPSHOT                          | Camel K operator image version used in operator installation. Value is retrieved from `pkg/util/defaults/defaults.go`                         |
| CAMEL_K_TEST_NO_OLM_OPERATOR_IMAGE      | docker.io/apache/camel-k:2.4.0-SNAPSHOT | Camel K operator image used in non OLM based operator installation.                                                                           |
| CAMEL_K_TEST_RUNTIME_VERSION            | 3.8.1                                   | Camel K runtime version used for the integrations. Value is retrieved from `pkg/util/defaults/defaults.go`                                    |
| CAMEL_K_TEST_BASE_IMAGE                 | eclipse-temurin:17                      | Camel K runtime base image used for the integrations. Value is retrieved from `pkg/util/defaults/defaults.go`                                 |
| CAMEL_K_TEST_TIMEOUT_SHORT              | 1                                       | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_TIMEOUT_MEDIUM             | 5                                       | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_TIMEOUT_LONG               | 15                                      | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_MAVEN_CA_PEM_PATH          | -                                       | Optional Maven certificate path.                                                                                                              |
| CAMEL_K_TEST_COPY_CATALOG               | true                                    | Enable/disable the optimization to copy the Camel Catalog from default operator namespace for each test namespace.                            |
| CAMEL_K_TEST_COPY_INTEGRATION_KITS      | true                                    | Enable/disable the optimization to copy integration kits from default operator namespace for each test namespace.                             |
| CAMEL_K_TEST_NS                         | -                                       | Custom test namespace name used to create temporary namespaces.                                                                               |
| CAMEL_K_TEST_MAKE_DIR                   | -                                       | Used in Helm and Kustomize install tests as Makefile root dir.                                                                                |
| CAMEL_K_TEST_MAKE_ARGS                  | -                                       | Used in Helm and Kustomize install tests as Makefile arguments.                                                                               |

## Structure of the directory

NOTE: dear contributor, please, keep this organization as clean as you can, updating any documentation if any change is done.

* builder
* common
* advanced
* install
* knative
* native
* telemetry
* yaks

### Builder

Contains a basic set of tests required to validate each builder strategy we offer. Ideally we don't want to test complex features but only a few test to validate any builder we offer is working correctly.

### Common

Full set of test to validate the main project feature. This test will assume the presence of a namespaced operator (installation provided by the same test execution suite). Insert here any test that has to validate any new feature required.

### Advanced

Additional set of test that cover the main common features but that requires some particular operator configuration. In this test suite you must take care of installing the operator as well.

### Install

Test suite that cover the different installation procedures we offer and any upgrade scenario.

### Knative

Test suite that cover the features associated with Knative. This test will assume the presence of a namespaced operator (installation provided by the same test execution suite) together with Knative operator configuration.

### Native

Test suite that cover the Quarkus Native build. As it is high resource consuming, we just validate that a native build for the supported DSLs is working.

### Telemetry

Test suite that cover the features associated with Telemetry feature. The test execution takes care of installing the required configuration.

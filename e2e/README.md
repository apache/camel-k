# Camel K End To End tests

This directory contains the suite of test that are run on a CI to ensure the stability of the product and no regression are introduced at each PR. The full documentation can be found at https://camel.apache.org/camel-k/next/contributing/e2e.html

## Structure of the directory

NOTE: dear contributor, please, keep this organization as clean as you can, updating any documentation if any change is done.

* builder
* common
* commonwithcustominstall
* install
* knative
* native
* telemetry
* yaks

### Builder

Contains a basic set of tests required to validate each builder strategy we offer. Ideally we don't want to test complex features but only a few test to validate any builder we offer is working correctly.

### Common

Full set of test to validate the main project feature. This test will assume the presence of a namespaced operator (installation provided by the same test execution suite). Insert here any test that has to validate any new feature required.

### Commonwithcustominstall

Additional set of test that cover the main common features but that requires some particular operator configuration. In this test suite you must take care of installing the operator as well.

### Install

Test suite that cover the different installation procedures we offer and any upgrade scenario.

### KNative

Test suite that cover the features associated with KNative. This test will assume the presence of a namespaced operator (installation provided by the same test execution suite) togheter with KNative operator configuration.

### Native

Test suite that cover the Quarkus Native build. As it is high resource consuming, we just validate that a native build for the supported DSLs is working.

### Telemetry

Test suite that cover the features associated with Telemetry feature. The test execution takes care of installing the required configuration.

### Yaks

Test suite that cover certain KNative features togheter with YAKS operator.

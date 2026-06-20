# Camel K End-To-End tests

This directory contains the suite of test that are run on a CI to ensure the stability of the product and no regression are introduced at each PR. The full documentation can be found at https://camel.apache.org/camel-k/next/contributing/e2e.html

## Environment variables

You can set some environment variables to change the behavior of the E2E test suite.

| Env                                     | Default                                 | Description                                                                                                                                   |
|-----------------------------------------|-----------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| CAMEL_K_TEST_TIMEOUT_SHORT              | 1                                       | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_TIMEOUT_MEDIUM             | 5                                       | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_TIMEOUT_LONG               | 15                                      | Customize the timeouts (in minutes) used in test assertions.                                                                                  |
| CAMEL_K_TEST_MAKE_DIR                   | -                                       | Used in Helm and Kustomize install tests as Makefile root dir.                                                                                |

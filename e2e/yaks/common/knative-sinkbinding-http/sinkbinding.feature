Feature: Camel K can run source HTTP endpoint in sinkbinding mode

  Background:
    Given Kubernetes resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Integration knative-service starts with no errors
    Given wait for condition=Ready on Kubernetes custom resource integration/rest2channel in integration.camel.apache.org/v1

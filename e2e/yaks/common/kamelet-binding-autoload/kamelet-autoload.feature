Feature: Camel K can load default secrets for Kamelets

  Background:
    Given Disable auto removal of Kamelet resources
    Given Disable auto removal of Kubernetes resources
    Given Camel-K resource polling configuration
      | maxAttempts          | 20   |
      | delayBetweenAttempts | 1000 |

  Scenario: KameletBinding can load default settings for Kamelet
    Given create Kubernetes service stub-service with target port 8080
    And bind Kamelet timer-source to uri http://stub-service.${YAKS_NAMESPACE}.svc.cluster.local/default
    When create KameletBinding binding
    Then KameletBinding binding should be available

 Scenario: Verify default binding
    Given HTTP server "stub-service"
    And HTTP server timeout is 600000 ms
    Then expect HTTP request body: default
    And receive POST /default
    And delete KameletBinding binding

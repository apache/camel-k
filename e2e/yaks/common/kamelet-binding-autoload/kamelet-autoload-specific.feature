Feature: Camel K can load specific secrets for Kamelets

  Background:
    Given Disable auto removal of Kamelet resources
    Given Disable auto removal of Kubernetes resources
    Given Camel-K resource polling configuration
      | maxAttempts          | 20   |
      | delayBetweenAttempts | 1000 |

  Scenario: KameletBinding can load specific settings for Kamelet
    Given create Kubernetes service stub-service-2 with target port 8081
    And bind Kamelet timer-source to uri http://stub-service-2.${YAKS_NAMESPACE}.svc.cluster.local/specific
    And KameletBinding source properties
      | id  | specific |
    When create KameletBinding binding-specific
    Then KameletBinding binding-specific should be available

 Scenario: Verify specific binding
    Given HTTP server "stub-service-2"
    And HTTP server timeout is 60000 ms
    Then expect HTTP request body: specific
    And receive POST /specific
    And delete KameletBinding binding-specific

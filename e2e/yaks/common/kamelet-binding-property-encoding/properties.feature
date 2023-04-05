Feature: Ensure that Kamelets support multiline configuration

  Background:
    Given Disable auto removal of Kamelet resources
    Given Disable auto removal of Kubernetes resources
    Given Camel K resource polling configuration
      | maxAttempts          | 60   |
      | delayBetweenAttempts | 3000 |

  Scenario: Wait for binding to start
    Given create Kubernetes service probe-service with target port 8080
    Then Camel K integration properties-binding should be running

  Scenario: Verify binding
    Given HTTP server "probe-service"
    And HTTP server timeout is 300000 ms
    Then expect HTTP request body
    """
    {
      "content": "thecontent",
      "key2": "val2"
    }
    """
    And expect HTTP request header: Content-Type="application/json;charset=UTF-8"
    And receive POST /events
    And delete KameletBinding properties-binding

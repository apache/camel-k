Feature: Kamelet may have no properties

  Background:
    Given Disable auto removal of Camel K resources
    Given Disable auto removal of Kamelet resources
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Create Kamelet
    Given create Kamelet no-props-source with flow
"""
from:
  uri: timer:tick
  steps:
  - set-body:
      constant: "Hello World"
  - to: "kamelet:sink"
"""
    Then Kamelet no-props-source should be available


  Scenario: Bind Kamelet to service
    Given create Kubernetes service greeting-service with target port 8080
    And bind Kamelet no-props-source to uri log:info
    When create KameletBinding no-props-source-uri
    Then KameletBinding no-props-source-uri should be available
    Then Camel K integration no-props-source-uri should be running
    Then Camel K integration no-props-source-uri should print Hello World

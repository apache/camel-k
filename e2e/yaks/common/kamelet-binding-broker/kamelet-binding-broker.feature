Feature: Camel K can bind Kamelets to the broker

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Sending event to the custom broker with KameletBinding
    Given Camel K integration logger-sink-binding-br is running
    Then Camel K integration logger-sink-binding-br should print message: Hello Custom Event from sample-broker

  Scenario: Remove resources
    Given delete Camel K integration timer-source-binding-br
    Given delete Camel K integration logger-sink-binding-br

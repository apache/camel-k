Feature: Camel K can bind Kamelets

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given Camel K integration logger-sink-binding-kb is running
    Then Camel K integration logger-sink-binding-kb should print message: Hello Kamelets

  Scenario: Remove resources
    Given delete Camel K integration timer-source-binding-kb
    Given delete Camel K integration logger-sink-binding-kb


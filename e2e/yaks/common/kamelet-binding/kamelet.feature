Feature: Camel K can bind Kamelets

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given Camel K integration logger-sink-binding is running
    Then Camel K integration logger-sink-binding should print message: Hello Kamelets

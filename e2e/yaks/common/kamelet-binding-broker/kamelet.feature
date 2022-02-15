Feature: Camel K can bind Kamelets to the broker

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Sending event to the broker with KameletBinding
    Given Camel K integration logger-sink-binding is running
    Then Camel K integration logger-sink-binding should print message: Hello Custom Event

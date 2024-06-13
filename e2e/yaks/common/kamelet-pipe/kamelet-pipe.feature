Feature: Camel K can bind Kamelets

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Running integration using a simple Kamelet with Pipes
    Given Camel K integration timer-source-pipe is running
    Given Camel K integration logger-sink-pipe is running
    Then Camel K integration logger-sink-pipe should print message: Hello Kamelets

  Scenario: Remove resources
    Given delete Camel K integration timer-source-pipe
    Given delete Camel K integration logger-sink-pipe


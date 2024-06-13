Feature: Kamelets with data types

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 200  |
      | delayBetweenAttempts | 4000 |

  Scenario: Kamelet event data type conversion
    Given load Pipe event-pipe.yaml
    Given Camel K integration event-pipe is running
    Then Camel K integration event-pipe should print BodyType: byte[], Body: Hello from Camel K!
    Then Camel K integration event-pipe should print BodyType: String, Body: Hello from Camel K!

  Scenario: Kamelet timer-to-log conversion
    Given load Pipe timer-to-log.yaml
    Given Camel K integration timer-to-log is running
    Then Camel K integration timer-to-log should print BodyType: byte[], Body: Hello from Camel K!
    Then Camel K integration timer-to-log should print BodyType: String, Body: Hello from Camel K!

  Scenario: Remove resources
    Given delete Pipe event-pipe
    Given delete Pipe timer-to-log

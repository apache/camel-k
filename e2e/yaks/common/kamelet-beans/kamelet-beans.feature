Feature: Kamelets can declare local beans

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Kamelets templates can use beans
    Given bind Kamelet beans-source to uri log:info
    When create Pipe binding
    Then Pipe binding should be available
    Then Camel K integration binding should be running
    Then Camel K integration binding should print Bean time is 0!

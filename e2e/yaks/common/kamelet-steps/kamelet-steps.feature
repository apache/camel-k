Feature: Pipe can have multiple processing steps

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Data is transformed by the steps
    Given Camel K integration steps-pipe is running
    Then Camel K integration steps-pipe should print Hello Apache Camel

  Scenario: Remove resources
    Given delete Camel K integration steps-pipe

Feature: KameletBindings can have multiple processing steps

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Data is transformed by the steps
    Given Camel K integration steps-binding is running
    Then Camel K integration steps-binding should print Hello Apache Camel

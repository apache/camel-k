Feature: Camel K can run Kamelets from default catalog

  Background:
    Given Camel-K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Integrations can use default catalog
    Given Camel-K integration logger is running
    Then Camel-K integration logger should print Camel K

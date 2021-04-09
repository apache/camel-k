Feature: Camel K can run Kamelets

  Background:
    Given Camel-K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Integrations can use multiple kamelets
    Given Camel-K integration source-sink is running
    Then Camel-K integration source-sink should print nice echo: Camel K

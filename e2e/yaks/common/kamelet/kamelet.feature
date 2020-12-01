Feature: Camel K can run Kamelets

  Scenario: Integrations can use multiple kamelets
    Given Camel-K integration source-sink is running
    Then Camel-K integration source-sink should print nice echo: Camel K

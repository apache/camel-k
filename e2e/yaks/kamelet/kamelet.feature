Feature: Camel K can run Kamelets

  Scenario: Integrations can use multiple kamelets
    Given integration source-sink is running
    Then integration source-sink should print nice echo: Camel K

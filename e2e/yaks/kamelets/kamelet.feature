Feature: Camel K can run Kamelets and bind them

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given integration logger is running
    Then integration logger should print Hello Kamelets

  Scenario: Integrations can use multiple kamelets
    Given integration source-sink is running
    Then integration source-sink should print nice echo: Camel K

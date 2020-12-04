Feature: Camel K can run Kamelets from default catalog

  Scenario: Integrations can use default catalog
    Given Camel-K integration logger is running
    Then Camel-K integration logger should print Camel K

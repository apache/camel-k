Feature: Camel K can run Kamelets from default catalog

  Scenario: Integrations can use default catalog
    Given integration logger is running
    Then integration logger should print Camel K

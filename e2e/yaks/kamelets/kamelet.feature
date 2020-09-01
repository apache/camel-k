Feature: Camel K can run Kamelets and bind them

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given integration logger is running
    Then integration logger should print Hello Kamelets

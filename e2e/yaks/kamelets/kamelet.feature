Feature: Camel K can run Kamelets

  Scenario: Running integration using a simple Kamelet
    Given integration usage is running
    Then integration usage should print Hello Kamelets

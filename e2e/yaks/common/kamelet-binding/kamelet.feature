Feature: Camel K can bind Kamelets

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given integration logger-sink-binding is running
    Then integration logger-sink-binding should print message: Hello Kamelets

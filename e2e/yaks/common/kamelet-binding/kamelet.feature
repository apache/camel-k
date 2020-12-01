Feature: Camel K can bind Kamelets

  Scenario: Running integration using a simple Kamelet with KameletBinding
    Given Camel-K integration logger-sink-binding is running
    Then Camel-K integration logger-sink-binding should print message: Hello Kamelets

  Scenario: Binding to a HTTP URI should use CloudEvents
    Given Camel-K integration display is running
    Then Camel-K integration display should print type: org.apache.camel.event
    Then Camel-K integration display should print Hello

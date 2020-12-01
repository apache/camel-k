Feature: Camel K can bind Kamelets to the broker

  Scenario: Sending event to the broker with KameletBinding
    Given Camel-K integration logger-sink-binding is running
    Then Camel-K integration logger-sink-binding should print message: Hello Custom Event

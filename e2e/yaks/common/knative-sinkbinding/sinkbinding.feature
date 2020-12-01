Feature: Camel K can run source in sinkbinding mode

  Scenario: Integration gets the message from the sinkbinding source
    Given Camel-K integration receiver is running
    Then Camel-K integration receiver should print HELLO SINKBINDING

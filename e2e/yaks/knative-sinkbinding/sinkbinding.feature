Feature: Camel K can run source in sinkbinding mode

  Scenario: Integration gets the message from the sinkbinding source
    Given integration receiver is running
    Then integration receiver should print HELLO SINKBINDING

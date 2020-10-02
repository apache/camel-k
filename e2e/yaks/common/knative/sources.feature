Feature: Camel K can run latest released Knative CamelSource

  Scenario: Integration gets the message from the source
    Given integration receiver is running
    Then integration receiver should print MagicString!

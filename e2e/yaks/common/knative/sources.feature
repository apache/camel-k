Feature: Camel K can run latest released Knative CamelSource

  Scenario: Integration gets the message from the source
    Given Camel-K integration receiver is running
    Then Camel-K integration receiver should print MagicString!

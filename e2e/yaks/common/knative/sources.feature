Feature: Camel K can run latest released Knative CamelSource

  Background:
    Given Camel-K resource polling configuration
      | maxAttempts          | 40   |
      | delayBetweenAttempts | 3000 |

  Scenario: Integration gets the message from the source
    Given Camel-K integration receiver is running
    Then Camel-K integration receiver should print MagicString!

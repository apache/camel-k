Feature: KameletBindings can have multiple processing steps

  Scenario: Data is transformed by the steps
    Given Camel-K integration steps-binding is running
    Then Camel-K integration steps-binding should print Hello Apache Camel

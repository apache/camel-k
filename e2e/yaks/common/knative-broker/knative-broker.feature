Feature: Camel K can correctly filter messages from broker

  Background:
    Given create Knative broker default
    Given Knative broker default is running
    Given Disable auto removal of Camel-K resources
    Given Disable auto removal of Kubernetes resources
    Given Camel-K resource polling configuration
      | maxAttempts          | 60   |
      | delayBetweenAttempts | 3000 |

  Scenario: Integration sends messages to the broker
    Given create Camel-K integration sender.groovy
    """
    from('timer:tick?period=1000')
      .setBody().constant('event-1')
      .to('knative:event/evt1')

    from('timer:tick?period=1000')
      .setBody().constant('event-2')
      .to('knative:event/evt2')

    from('timer:tick?period=1000')
      .setBody().constant('event-all')
      .to('knative:event')
    """
    Then Camel-K integration sender should be running


  Scenario: Integration receives the correct messages from the broker
    Given create Camel-K integration receiver.groovy
    """
    from('knative:event/evt1')
      .log('From evt1: $simple{body}')

    from('knative:event/evt2')
      .log('From evt2: $simple{body}')

    from('knative:event')
      .log('From all: $simple{body}')
    """
    Then Camel-K integration receiver should be running
    And Camel-K integration receiver should print From evt1: event-1
    And Camel-K integration receiver should print From evt2: event-2
    And Camel-K integration receiver should print From all: event-1
    And Camel-K integration receiver should print From all: event-2
    And Camel-K integration receiver should print From all: event-all
    And Camel-K integration receiver should not print From evt1: event-2
    And Camel-K integration receiver should not print From evt1: event-all
    And Camel-K integration receiver should not print From evt2: event-1
    And Camel-K integration receiver should not print From evt2: event-all
    And delete Camel-K integration sender
    And delete Camel-K integration receiver

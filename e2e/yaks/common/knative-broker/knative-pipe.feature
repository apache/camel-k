Feature: Pipes connecting with Knative broker

  Background:
    Given create Knative broker default
    Given Knative broker default is running
    Given Camel K resource polling configuration
      | maxAttempts          | 60   |
      | delayBetweenAttempts | 3000 |

  Scenario: Pipes exchanging events with the broker
    # Pipe pushing events to the broker
    Given load Pipe event-source-pipe.yaml
    Then Camel K integration event-source-pipe should be running

    # Pipe receives given event type from the broker
    Given load Pipe log-sink-pipe.yaml
    Then Camel K integration log-sink-pipe should be running
    And Camel K integration log-sink-pipe should print Hello this is event-1!

    # Pipe receives all events from the broker
    Given load Pipe no-filter-pipe.yaml
    Then Camel K integration no-filter-pipe should be running
    And Camel K integration no-filter-pipe should print Hello this is event-1!

    # Pipe receives events with source filter
    Given load Pipe source-filter-pipe.yaml
    Then Camel K integration source-filter-pipe should be running
    And Camel K integration source-filter-pipe should print Hello this is event-1!

  Scenario: Remove resources
    Given delete Camel K integration event-source-pipe
    Given delete Camel K integration log-sink-pipe
    Given delete Camel K integration source-filter-pipe
    Given delete Camel K integration no-filter-pipe

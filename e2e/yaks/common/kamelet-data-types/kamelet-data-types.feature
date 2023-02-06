Feature: Kamelets with data types

  Background:
    Given Camel K resource polling configuration
      | maxAttempts          | 200  |
      | delayBetweenAttempts | 4000 |

  Scenario: Kamelet event data type conversion
    Given variables
    | outputFormat | binary |
    | inputFormat  | string |
    Given load KameletBinding event-binding.yaml
    Given Camel K integration event-binding is running
    Then Camel K integration event-binding should print BodyType: byte[], Body: Hello from Camel K!
    Then Camel K integration event-binding should print BodyType: String, Body: Hello from Camel K!

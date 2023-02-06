Feature: Camel K can serve metrics to Prometheus

  Background: Prepare Thanos URL
    Given URL: https://thanos-querier.openshift-monitoring:9091

  Scenario: Integration gets the message from the timer
    Given Camel K integration metrics is running
    Then Camel K integration metrics should print Successfully processed
    Then sleep 120000 ms

  Scenario: Thanos is able to serve custom microprofile annotation metrics
    Given HTTP request header Authorization is "Bearer ${openshift.token}"
    When send GET /api/v1/query?query=application_camel_k_example_metrics_attempt_total
    Then verify HTTP response expressions
      | $.status                         | success                                           |
      | $.data.result[0].metric.__name__ | application_camel_k_example_metrics_attempt_total |
      | $.data.result[0].metric.pod      | @startsWith('metrics')@                           |
      | $.data.result[0].value[1]        | @isNumber()@                                      |
    And receive HTTP 200

  Scenario: Thanos is able to serve custom camel microprofile metrics
    Given HTTP request header Authorization is "Bearer ${openshift.token}"
    When send GET /api/v1/query?query=application_camel_k_example_metrics_error_total
    Then verify HTTP response expressions
      | $.status                         | success                                         |
      | $.data.result[0].metric.__name__ | application_camel_k_example_metrics_error_total |
      | $.data.result[0].metric.pod      | @startsWith('metrics')@                         |
      | $.data.result[0].value[1]        | @isNumber()@                                    |
    And receive HTTP 200

  Scenario: Thanos is able to serve integration build metrics from Operator
    Given HTTP request header Authorization is "Bearer ${openshift.token}"
    When send GET /api/v1/query?query=camel_k_build_duration_seconds_sum
    Then verify HTTP response expressions
      | $.status                                                                    | success                            |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}' && @.metric.result == 'Succeeded')].metric.__name__ | camel_k_build_duration_seconds_sum |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}' && @.metric.result == 'Succeeded')].metric.pod      | @startsWith('camel-k-operator')@   |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}' && @.metric.result == 'Succeeded')].value[1]        | @greaterThan(10)@                  |
    And receive HTTP 200

  Scenario: Thanos is able to serve integration readiness metrics from Operator
    Given HTTP request header Authorization is "Bearer ${openshift.token}"
    When send GET /api/v1/query?query=camel_k_integration_first_readiness_seconds_sum
    Then verify HTTP response expressions
      | $.status                                                                    | success                                         |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}')].metric.__name__ | camel_k_integration_first_readiness_seconds_sum |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}')].metric.pod      | @startsWith('camel-k-operator')@                |
      | $.data.result[?(@.metric.namespace == '${YAKS_NAMESPACE}')].value[1]        | @greaterThan(5)@                               |
    And receive HTTP 200

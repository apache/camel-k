Feature: Alerts from Camel-K are propagated to Openshift Prometheus

  Background: Prepare Thanos-ruler URL
    Given URL: https://thanos-ruler.openshift-user-workload-monitoring:9091

  Scenario: Integration gets the message from the timer
    Given Camel-K integration metrics is running
    Then Camel-K integration metrics should print Successfully processed
    Then sleep 120000 ms

  Scenario: Thanos-ruler is able to serve alerts based on metrics from Operator
    Given HTTP request header Authorization is "Bearer ${openshift.token}"
    When send GET /api/v1/rules
    Then verify HTTP response expressions
      | $..rules[?(@.labels.namespace == '${YAKS_NAMESPACE}' && @.state == 'firing')].name | CamelKBuildFailure |
    And receive HTTP 200
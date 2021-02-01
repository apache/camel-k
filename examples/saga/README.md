# Saga Camel K examples

This example is from [camel-saga-quickstart](https://github.com/nicolaferraro/camel-saga-quickstart/) adapted to work with camel-k.

Make sure Camel K is installed in your namespace, or execute the following command to install it:

```
kamel install
```

* Start the lra-coordinator by using the `oc` or `kubectl` tool:
```
kubectl apply -f lra-coordinator.yaml
```

* Start the three demo services
```
kamel run -d camel-lra Payment.java
kamel run -d camel-lra Flight.java
kamel run -d camel-lra Train.java
```

* Start the saga application
```
kamel run -d camel-lra Saga.java
```

Then you can use ```kamel logs saga``` to check the output of the transactions.

Focusing on one of the services, e.g. the flight service, you will notice that when unexpected events are found,
the operation is subsequently cancelled, e.g.:

E.g. running:
```
kamel logs flight
```

Possible workflow:
```
flight-7c8df48b88-6pzwt integration 2020-03-02 10:56:30.148 INFO  [default-workqueue-2] route2 - Buying flight #18
flight-7c8df48b88-6pzwt integration 2020-03-02 10:56:30.165 ERROR [XNIO-1 I/O-1] DefaultErrorHandler - Failed delivery for (MessageId: ID-flight-7c8df48b88-6pzwt-1583146351094-0-106 on ExchangeId: ID-flight-7c8df48b88-6pzwt-1583146351094-0-105). Exhausted after delivery attempt: 1 caught: org.apache.camel.http.common.HttpOperationFailedException: HTTP operation failed invoking http://payment/api/pay?bridgeEndpoint=true&type=flight with statusCode: 500
...
# after stacktrace
...
flight-7c8df48b88-6pzwt integration 2020-03-02 10:56:30.256 INFO  [XNIO-2 task-6] route1 - Flight purchase #18 has been cancelled
flight-7c8df48b88-6pzwt integration 2020-03-02 10:56:35.150 INFO  [default-workqueue-3] route2 - Buying flight #19
flight-7c8df48b88-6pzwt integration 2020-03-02 10:56:35.197 INFO  [XNIO-1 I/O-1] route2 - Payment for flight #19 done
```

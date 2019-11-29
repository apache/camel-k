# Saga example
This example is from [camel-saga-quickstart](https://github.com/nicolaferraro/camel-saga-quickstart/) and could work with the camel-k.

* Start the lra-coordinator
```
oc create -f lra-coordinator.yaml
```
* Start the three demo services
```
kamel run -t container.service-port=8080 --dependency=camel-rest --dependency=camel-undertow --dependency=camel-lra examples/saga/Flight.java
kamel run -t container.service-port=8080 --dependency=camel-rest --dependency=camel-undertow --dependency=camel-lra examples/saga/Train.java
kamel run -t container.service-port=8080 --dependency=camel-rest --dependency=camel-undertow --dependency=camel-lra examples/saga/Payment.java
```
* Start the saga application
```
kamel run -t container.service-port=8080 --dependency=camel-rest --dependency=camel-undertow --dependency=camel-lra examples/saga/Saga.java
```
Then you can use ```kamel log saga``` to check the output of the transactions.
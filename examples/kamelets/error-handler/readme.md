# Kamelets Binding Error Handler example
This example shows how to create a simple _source_ `Kamelet` which sends periodically events (and certain failures). The events are consumed by a log _sink_ in a `KameletBinding`. With the support of the `ErrorHandler` we will be able to redirect all errors to a `Dead Letter Channel` _error-handler_ `Kamelet`.

## Incremental ID Source Kamelet
First of all, you must install the _incremental-id-source_ Kamelet defined in `incremental-id-source.kamelet.yaml` file. This source will emit events every second with an autoincrement counter that will be forced to fail when the number 0 is caught. With this trick, we will simulate possible event faults.
```
$ kubectl apply -f incremental-id-source.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME                            PHASE
incremental-id-source           Ready
```
## Log Sink Kamelet
Now it's the turn of installing the log-sink_ Kamelet defined in `log-sink.kamelet.yaml` file:
```
$ kubectl apply -f log-sink.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME                    PHASE
log-sink                Ready
incremental-id-source   Ready
```
## Error handler Kamelet
We finally install an error handler as specified in `error-handler.kamelet.yaml` file. This is a simple logger, but you can use any endpoint to collect and store the failing events.
```
$ kubectl apply -f error-handler.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME                    PHASE
error-handler           Ready
log-sink                Ready
incremental-id-source   Ready
```
## Error Handler Kamelet Binding
We can create a `KameletBinding` which is started by the _incremental-id-source_ `Kamelet` and log events to _log-sink_ `Kamelet`. As this will sporadically fail, we can configure an _errorHandler_ with the _error-handler_ `Kamelet` as **Dead Letter Channel**. We can declare it as in `kamelet-binding-error-handler.yaml` file:
```
...
  errorHandler:
    type: dead-letter-channel
    endpoint:
      ref:
        kind: Kamelet
        apiVersion: camel.apache.org/v1alpha1
        name: error-handler
      properties:
        message: "ERROR!"
```
Execute the following command to start the `Integration`:
```
kubectl apply -f kamelet-binding-error-handler.yaml
```
As soon as the `Integration` starts, it will log the events on the `ok` log channel and errors on the `error` log channel:
```
[1] 2021-04-21 13:03:43,773 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #8]
[1] 2021-04-21 13:03:44,774 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #9]
[1] 2021-04-21 13:03:45,898 INFO  [error-sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: ERROR!]
[1] 2021-04-21 13:03:46,775 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #11]
```
This example is useful to guide you through the configuration of an error handler. In a production environment you will likely configure the error handler `Kamelet` pointing to a persistent queue.
# Kamelets Binding Error Handler example
This example shows how to create a simple _source_ `kamelet` bound to a log _sink_ in a `KameletBinding`. With the support of the `ErrorHandler` we will be able to redirect all errors to a `Dead Letter Channel` _error-handler_ `Kamelet`.

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
## Error handler  Kamelet
We finally install an error handler as a dead letter channel as specified in `error-handler.kamelet.yaml` file. You can specify any kind of error handler as expected by Apache Camel runtime.
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
We can create a `KameletBinding` which is triggered by the _incremental-id-source_ `Kamelet` and log events to _log-sink_ `Kamelet`. As this will sporadically fail, we can configure an _errorHandler_ as defined in `kamelet-binding-error-handler.yaml` file:
```
...
  errorHandler:
    ref:
      kind: Kamelet
      apiVersion: camel.apache.org/v1alpha1
      name: error-handler
```
Execute the following command to start the `Integration`:
```
kubectl apply -f kamelet-binding-error-handler.yaml
```
As soon as the `Integration` starts, it will log the events on the `ok` log channel and errors on the `error` log channel:
```
[1] 2021-04-13 10:33:38,375 INFO  [ok] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #8]
[1] 2021-04-13 10:33:39,376 INFO  [ok] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #9]
[1] 2021-04-13 10:33:40,409 INFO  [error] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, Headers: {firedTime=Tue Apr 13 10:33:40 GMT 2021}, BodyType: String, Body: Producing message #10, CaughtExceptionType: org.apache.camel.CamelExecutionException, CaughtExceptionMessage: Exception occurred during execution on the exchange: Exchange[]]
[1] 2021-04-13 10:33:41,375 INFO  [ok] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #11]
```
This example is useful to guide you through the configuration of an error handler. In a production environment you will likely configure the error handler with a `Dead Letter Channel` pointing to a persistent queue.
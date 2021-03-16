# Kamelets Binding Error Handler example
This example shows how to create a simple _timer-source_ `kamelet` bound to a failing sink in a `KameletBinding`. With the support of the `ErrorHandler` we will be able to redirect all errors to a `Dead Letter Channel` _error-sink_ `Kamelet`.

## Timer Source Kamelet
First of all, you must install the _timer-source_ Kamelet defined in `timer-source.kamelet.yaml` file:
```
$ kubectl apply -f timer-source.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME                   PHASE
timer-source           Ready
```
## Error Sink Kamelet
Now it's the turn of installing the _error-sink_ Kamelet defined in `error-sink.kamelet.yaml` file:
```
$ kubectl apply -f error-sink.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME           PHASE
error-sink     Ready
timer-source   Ready
```
## Error Handler Kamelet Binding
We can create a `KameletBinding` which is triggered by the _timer-source_ `Kamelet` and define a fake sink URI in order to make it fail on purpose. Then we can configure an _errorHandler_ as defined in `kamelet-binding-error-handler.yaml` file:
```
  errorHandler:
    ref:
      kind: Kamelet
      apiVersion: camel.apache.org/v1alpha1
      name: log-sink
    properties:
     defaultMessage: "ERROR ERROR!" 
```
Execute the following command to start the `Integration`:
```
kubectl apply -f kamelet-binding-error-handler.yaml
```
As soon as the `Integration` starts, it will log a message error for every failure it will print:
```
[1] 2021-03-16 15:11:33,099 INFO  [error-sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: ERROR ERROR!]
[1] 2021-03-16 15:11:33,988 INFO  [error-sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: ERROR ERROR!]
[1] 2021-03-16 15:11:34,988 INFO  [error-sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: ERROR ERROR!]
```
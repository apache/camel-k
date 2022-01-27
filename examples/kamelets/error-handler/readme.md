# Kamelets Binding Error Handler example
This example shows how to create a simple _source_ `Kamelet` which sends periodically events (and certain failures). The events are consumed by a log _sink_ in a `KameletBinding`. With the support of the `ErrorHandler` we will be able to redirect all errors to a `Sink` _error-handler_ `Kamelet` whose goal is to store the events in a `Kafka` topic and provide a nice log notifying us about the error happened.

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
We finally install an error handler as specified in `error-handler.kamelet.yaml` file. Let's have a look at how it is configured:

```
apiVersion: camel.apache.org/v1alpha1
kind: Kamelet
metadata:
  name: error-handler
spec:
  definition:
    ...
    properties:
      kafka-brokers:
        ...  
      kafka-topic:
        ...
      kafka-service-account-id:
        ...  
      kafka-service-account-secret:
        ...              
      log-message:
        ...    
  template:
    from:
      uri: kamelet:source
      steps:
      # First step: send to the DLC for future processing
      - to:
          uri: kafka:{{kafka-topic}}
          parameters:
            brokers: "{{kafka-brokers}}"
            security-protocol: SASL_SSL
            sasl-mechanism: PLAIN
            sasl-jaas-config: "org.apache.kafka.common.security.plain.PlainLoginModule required username={{kafka-service-account-id}} password={{kafka-service-account-secret}};"
      # Log an error message to notify about the failure
      - set-body:
          constant: "{{log-message}} - worry not, the event is stored in the DLC"
      - to: "log:error-sink"
```

We first send the errored event to a kafka topic, and then, we send a simple notification message to output, just to let the user know that some issue happened. Let's apply it:

```
$ kubectl apply -f error-handler.kamelet.yaml
```
You can check the newly created `kamelet` listing the kamelets available:
```
$ kubectl get kamelets

NAME                    PHASE
error-handler           Ready
log-sink                Ready
incremental-id-source   Ready
```
## Error Handler Kamelet Binding
We can now create a `KameletBinding` which is started by the _incremental-id-source_ `Kamelet` and log events to _log-sink_ `Kamelet`. As this will sporadically fail, we can configure an _errorHandler_ with the _error-handler_ `Kamelet` as **Sink**. We want to configure also some redelivery policies (1 retry, with a 2000 milliseconds delay). We can declare it as in `kamelet-binding-error-handler.yaml` file:
```
...
  errorHandler:
    sink:
      endpoint:
        ref:
          kind: Kamelet
          apiVersion: camel.apache.org/v1alpha1
          name: error-handler
        properties:
          message: "ERROR!"
      parameters:
        maximumRedeliveries: 1
        redeliveryDelay: 2000
```
Execute the following command to start the `Integration`:
```
kubectl apply -f kamelet-binding-error-handler.yaml
```
As soon as the `Integration` starts, it will log the events on the `ok` log channel and errors on the `error` log channel:
```
[1] 2021-04-29 08:35:08,875 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #49]
[1] 2021-04-29 08:35:11,878 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #51]
[1] 2021-04-29 08:35:12,088 INFO  [error-sink] (Camel (camel-1) thread #9 - KafkaProducer[my-first-test]) Exchange[ExchangePattern: InOnly, BodyType: String, Body: ERROR! - worry not, the event is stored in the DLC]
[1] 2021-04-29 08:35:12,877 INFO  [sink] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Producing message #52]
```

### Recover the errors from DLC

If you're curious to know what was going on in the DLC side, you can use the example you found in [kafka sasl ssl consumer](../kafka/sasl_ssl/):

```
kamel run --config secret:kafka-props SaslSSLKafkaConsumer.java --dev
...
[1] 2021-04-29 08:57:08,636 INFO  [org.apa.kaf.com.uti.AppInfoParser] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Kafka commitId: 448719dc99a19793
[1] 2021-04-29 08:57:08,636 INFO  [org.apa.kaf.com.uti.AppInfoParser] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Kafka startTimeMs: 1619686628635
[1] 2021-04-29 08:57:08,637 INFO  [org.apa.cam.com.kaf.KafkaConsumer] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Subscribing my-first-test-Thread 0 to topic my-first-test
...
[1] 2021-04-29 08:35:02,894 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #40
[1] 2021-04-29 08:35:12,995 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #50
[1] 2021-04-29 08:35:22,879 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #60
...
´´´

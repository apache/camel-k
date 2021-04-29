# Kafka Camel K SASL SSL example

This example shows how Camel K can be used to connect to a generic Kafka broker using SASL SSL authentication mechanism. Edit the `application` properties file with the proper values to be able to authenticate a Kafka Broker and use any Topic.

For convenience create a secret to contain the sensitive properties in the `application.properties` file:

```
kubectl create secret generic kafka-props --from-file application.properties
```

Finally run this sample using the command:

```
kamel run --secret kafka-props SaslSSLKafkaConsumer.java --dev
```

A consumer will start logging the events found in the Topic:

```
[1] 2021-04-29 08:57:16,894 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #1096
[1] 2021-04-29 08:57:18,995 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #1097
[1] 2021-04-29 08:57:20,879 INFO  [FromKafka] (Camel (camel-1) thread #0 - KafkaConsumer[my-first-test]) Producing message #1098
```
# Kafka Camel K SASL SSL example

This example shows how Camel K can be used to connect to a generic Kafka broker using SASL SSL authentication mechanism. Edit the `application` properties file with the proper values to be able to authenticate a Kafka Broker and use any Topic.

## Prerequisite

You have a Kafka broker available on the cluster, or anywhere else accessible from the cluster. You will need to edit the `application.properties` file setting a "kafka broker", a "SASL username" and a "SASL password". 

## Secret Configuration

For convenience create a secret to contain the sensitive properties in the `application.properties` file:

```
kubectl create secret generic kafka-props --from-file application.properties
```

## Run a producer

At this stage, run a producer integration to fill the topic with a message, every 10 seconds:

```
kamel run --config secret:kafka-props SaslSSLKafkaProducer.java --dev
```

The producer will create a new message every 10 seconds, push into the topic and log some information.

```
[2] 2021-05-06 08:48:11,854 INFO  [FromTimer2Kafka] (Camel (camel-1) thread #1 - KafkaProducer[test]) Message correctly sent to the topic!
[2] 2021-05-06 08:48:11,854 INFO  [FromTimer2Kafka] (Camel (camel-1) thread #3 - KafkaProducer[test]) Message correctly sent to the topic!
[2] 2021-05-06 08:48:11,973 INFO  [FromTimer2Kafka] (Camel (camel-1) thread #5 - KafkaProducer[test]) Message correctly sent to the topic!
[2] 2021-05-06 08:48:12,970 INFO  [FromTimer2Kafka] (Camel (camel-1) thread #7 - KafkaProducer[test]) Message correctly sent to the topic!
[2] 2021-05-06 08:48:13,970 INFO  [FromTimer2Kafka] (Camel (camel-1) thread #9 - KafkaProducer[test]) Message correctly sent to the topic!
```


## Run a consumer

Now, open another shell and run the consumer integration using the command:

```
kamel run --config secret:kafka-props SaslSSLKafkaConsumer.java --dev
```

A consumer will start logging the events found in the Topic:

```
[1] 2021-05-06 08:51:08,991 INFO  [FromKafka2Log] (Camel (camel-1) thread #0 - KafkaConsumer[test]) Message #8
[1] 2021-05-06 08:51:10,065 INFO  [FromKafka2Log] (Camel (camel-1) thread #0 - KafkaConsumer[test]) Message #9
[1] 2021-05-06 08:51:10,991 INFO  [FromKafka2Log] (Camel (camel-1) thread #0 - KafkaConsumer[test]) Message #10
[1] 2021-05-06 08:51:11,991 INFO  [FromKafka2Log] (Camel (camel-1) thread #0 - KafkaConsumer[test]) Message #11
```
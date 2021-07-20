# Camel-k Larky Compute Service Routes

Proto files can be found [here](https://github.com/moehajj/camel-k-proto-example).

## Running the Grpc service

Run the route.
```shell
kamel run --dev \
        -d mvn:com.camel.k.example:camel-k-proto-example:1.0-SNAPSHOT \
        -d mvn:org.apache.camel:camel-grpc:3.9.0 \
        -d mvn:org.apache.camel:camel-componentdsl:3.9.0 \
        ExampleGrpcConsumerRoute.java
```

## Running the producer

To test the service, run the following producer.
```shell
kamel run --dev \
        -d mvn:com.camel.k.example:camel-k-proto-example:1.0-SNAPSHOT \
        -d mvn:org.apache.camel:camel-grpc:3.9.0 \
        -d mvn:org.apache.camel:camel-componentdsl:3.9.0 \
        ExampleGrpcProducerRoute.java
```
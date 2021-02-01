# Kafka Camel K examples

This example shows how Camel K can be used to connect to a Kafka broker.

To run this example first set-up Kafka on your k8s cluster.
A convenient way to do so is by using the Strimzi project, if you are using minikube follow these instructions at https://strimzi.io/quickstarts/minikube/

For convenience create a configmap to contain the properties:
```
kubectl create configmap kafka.props  --from-file=examples/kafka/application.properties
```

IMPORTANT: The kafka.host value in application.properties needs to be set to the CLUSTER-IP address of the my-cluster-kafka-bootstrap service in the kafka namespace:
 `kubectl get services -n kafka | grep my-cluster-kafka-bootstrap | awk '/[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/ {print $3}'`

Finally run this sample using the command:
```
./kamel run examples/kafka/SampleKafkaConsumer.java --configmap=kafka.props
```

To create messages to be read use the producer command from the Strimzi page:
```
kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.11.1-kafka-2.1.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic my-topic
```
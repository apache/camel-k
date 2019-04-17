/* To run this example first set-up Kafka. If using minikube follow these instructions:
    https://strimzi.io/quickstarts/minikube/
    Create a configmap to contain the properties e.g.
      kubectl create configmap kafka.props  --from-file=examples/application.properties
      Note: The kafka.host value in application.properties needs to be set to the CLUSTER-IP address of the my-cluster-kafka-bootstrap service in the kafka namespace
        e.g. kubectl get services -n kafka | grep my-cluster-kafka-bootstrap | awk '/[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/ {print $3}'
    Run this sample using the command:
      kamel run examples/Sample_Kafka_Consumer.java --configmap=kafka.props
    To create messages to be read use the producer commanf from the Strimzi page e.g.
      kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.11.1-kafka-2.1.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic my-topic
*/

import org.apache.camel.builder.RouteBuilder;

public class Sample_Kafka_Consumer extends RouteBuilder {
  @Override
  public void configure() throws Exception {
	log.info("About to start route: Kafka Server -> Log ");
	from("kafka:{{consumer.topic}}?brokers={{kafka.host}}:{{kafka.port}}"
             + "&maxPollRecords={{consumer.maxPollRecords}}"
             + "&consumersCount={{consumer.consumersCount}}"
             + "&seekTo={{consumer.seekTo}}"
             + "&groupId={{consumer.group}}")
             .routeId("FromKafka")
             .log("${body}");
  }
}

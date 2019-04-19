import org.apache.camel.builder.RouteBuilder;

public class SampleKafkaConsumer extends RouteBuilder {
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

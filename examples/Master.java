// camel-k: language=java

import org.apache.camel.builder.RouteBuilder;

/**
 * This example shows how to start a route on a single instance of the integration.
 * Increase the number of replicas to see it in action (the route will be started on a single pod only).
 */
public class Master extends RouteBuilder {
  @Override
  public void configure() throws Exception {

      // Write your routes here, for example:
      from("master:lock:timer:master?period=1s")
        .setBody()
          .simple("This message is printed by a single pod, even if you increase the number of replicas!")
        .to("log:info");

  }
}

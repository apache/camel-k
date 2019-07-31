package kamel;

import org.apache.camel.builder.RouteBuilder;

public class Routes extends RouteBuilder {

  @Override
  public void configure() throws Exception {
	  from("timer:tick")
		.setBody(constant("Hello! Camel K rocks!!!"))
		.to("log:info");
  }

}
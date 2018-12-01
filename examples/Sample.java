import org.apache.camel.builder.RouteBuilder;

public class Sample extends RouteBuilder {
  @Override
  public void configure() throws Exception {
	  from("timer:tick")
    .setBody(constant("-\n             r\n             o\n             c\nHello! Camel K\n             s\n             !\n"))
		.to("log:info?skipBodyLineSeparator=false");
  }
}
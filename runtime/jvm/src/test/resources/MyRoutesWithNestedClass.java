import org.apache.camel.Exchange;
import org.apache.camel.Processor;
import org.apache.camel.builder.RouteBuilder;

public class MyRoutesWithNestedClass extends RouteBuilder {
    @Override
    public void configure() throws Exception {
        Processor toUpper = new Processor() {
            @Override
            public void process(Exchange exchange) throws Exception {
                String body = exchange.getIn().getBody(String.class);
                body = body.toUpperCase();

                exchange.getOut().setBody(body);
            }
        };

        from("timer:tick")
            .setBody().constant("test")
            .process(toUpper)
            .to("log:info");
    }
}
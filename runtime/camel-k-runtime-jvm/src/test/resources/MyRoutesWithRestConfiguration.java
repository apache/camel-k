import org.apache.camel.builder.RouteBuilder;

public class MyRoutesWithRestConfiguration extends RouteBuilder {
    @Override
    public void configure() throws Exception {
        restConfiguration()
            .component("restlet")
            .host("localhost")
            .port("8080");

        from("timer:tick")
            .to("log:info");
    }
}
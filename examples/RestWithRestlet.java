//
// To run this integrations use:
//
//     kamel run --name=rest-with-restlet --dependency=camel-restlet examples/RestWithRestlet.java
//
public class RestWithRestlet extends org.apache.camel.builder.RouteBuilder {
    @Override
    public void configure() throws Exception {
        restConfiguration()
            .component("restlet")
            .host("0.0.0.0")
            .port("8080");

        rest()
            .get("/hello")
            .to("direct:hello");

        from("direct:hello")
            .transform().simple("Hello World");
    }
}
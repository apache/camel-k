import org.apache.camel.builder.RouteBuilder;

public class CaffeineCacheSample extends RouteBuilder {
  @Override
  public void configure() throws Exception {
                from("timer:tick")
                    .setBody(constant("Hello"))
                    .setHeader("CamelCaffeineAction", constant("PUT"))
                    .setHeader("CamelCaffeineKey", constant("1"))
                    .toF("caffeine-cache://%s", "test")
                    .log("Result of Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} is: ${body}")
                    .setBody(constant(null))
                    .setHeader("CamelCaffeineAction", constant("GET"))
                    .setHeader("CamelCaffeineKey", constant("1"))
                    .toF("caffeine-cache://%s", "test")
                    .log("Result of Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} is: ${body}")
                    .setBody(constant(null))
                    .setHeader("CamelCaffeineAction", constant("INVALIDATE"))
                    .setHeader("CamelCaffeineKey", constant("1"))
                    .log("Invalidating entry with key ${header.CamelCaffeineKey}")
                    .toF("caffeine-cache://%s", "test")
                    .setHeader("CamelCaffeineAction", constant("GET"))
                    .setHeader("CamelCaffeineKey", constant("1"))
                    .toF("caffeine-cache://%s", "test")
                    .log("The Action ${header.CamelCaffeineAction} with key ${header.CamelCaffeineKey} has result? ${header.CamelCaffeineActionHasResult}");
  }
}

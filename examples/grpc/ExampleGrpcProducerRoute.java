import com.camel.k.example.ExampleRequest;
import com.camel.k.example.ExampleResponse;

import org.apache.camel.Message;
import org.apache.camel.builder.RouteBuilder;

public class ExampleGrpcProducerRoute extends RouteBuilder {

    @Override
    public void configure() throws Exception {
        from("timer:tick")
                .process(exchange -> {
                    final Message message = exchange.getMessage();

                    String requestMessage = "Can you hear me from camel-k?";
                    int id = 226355;

                    String responseMessage = "I received " + requestMessage;
                    final ExampleRequest request = ExampleRequest.newBuilder()
                            .setRequestMessage(responseMessage)
                            .setRequestId(id)
                            .build();

                    message.setBody(request);
                })
                .to("grpc://example-grpc:9000/com.camel.k.example.ExampleService?method=run&synchronous=true")
                .log("Response received: ${body}");
    }
}
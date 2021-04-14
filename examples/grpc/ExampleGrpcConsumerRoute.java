import com.camel.k.example.ExampleRequest;
import com.camel.k.example.ExampleResponse;

import org.apache.camel.Message;
import org.apache.camel.builder.RouteBuilder;

public class ExampleGrpcConsumerRoute extends RouteBuilder {

    @Override
    public void configure() throws Exception {
        fromF("grpc://0.0.0.0:9000/com.camel.k.example.ExampleService?synchronous=true")
                .process(exchange -> {
                    final Message message = exchange.getMessage();
                    final ExampleRequest request = message.getBody(ComputeRequest.class);

                    String requestMessage = request.getRequestMessage();
                    int id = request.getRequestId();

                    String responseMessage = "I received " + requestMessage;
                    final ExampleResponse response = ExampleResponse.newBuilder()
                            .setResponseMessage(responseMessage)
                            .setResponseId(id)
                            .build();

                    message.setBody(response);
                });
    }
}
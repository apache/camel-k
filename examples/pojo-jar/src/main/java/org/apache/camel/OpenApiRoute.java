// camel-k: name=pojo-jar
// camel-k: dependency=camel-jackson
// camel-k: dependency=camel-openapi-java
// camel-k: dependency=mvn:org.projectlombok:lombok:1.18.22
// camel-k: resource=file:../../../../../../target/pojo-jar-1.0.0.jar
// camel-k: trait=jvm.classpath=/etc/camel/resources/pojo-jar-1.0.0.jar

package org.apache.camel;

import lombok.extern.slf4j.Slf4j;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.rest.RestBindingMode;
import org.apache.camel.model.rest.RestParamType;

@Slf4j
public class OpenApiRoute extends RouteBuilder {
    @Override
    public void configure() {
        getContext().getRegistry().bind("userService", new UserService());

        restConfiguration()
                .component("platform-http")
                .bindingMode(RestBindingMode.json)
                .dataFormatProperty("prettyPrint", "true")
                .apiContextPath("/api-doc")
                .apiProperty("api.title", "User API").apiProperty("api.version", "1.2.3")
                .apiProperty("cors", "true")
                .clientRequestValidation(true);

        rest("/users").description("User rest service")
                .consumes("application/json").produces("application/json")

                .get("/{id}").description("Find user by id").outType(User.class)
                .param().name("id").type(RestParamType.path).description("The id of the user to get").dataType("integer").endParam()
                .responseMessage().code(200).message("The user").endResponseMessage()
                .to("bean:userService?method=getUser(${header.id})")

                .put().description("Updates or create a user").type(User.class)
                .param().name("body").type(RestParamType.body).description("The user to update or create").required(true).endParam()
                .responseMessage().code(200).message("User created or updated").endResponseMessage()
                .to("bean:userService?method=updateUser")

                .get().description("Find all users").outType(User[].class)
                .responseMessage().code(200).message("All users").endResponseMessage()
                .to("bean:userService?method=listUsers")

                .get("/{id}/departments/{did}").description("Find all users").outType(User[].class)
                .param().name("id").type(RestParamType.path).description("The id of the user to get").dataType("integer").endParam()
                .param().name("did").type(RestParamType.path).description("The id of the department to get").dataType("integer").required(true).endParam()
                .responseMessage().code(200).message("All users").endResponseMessage()
                .to("bean:userService?method=listUsers");
    }
}

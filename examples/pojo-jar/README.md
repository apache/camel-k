# camel-k-jar
This example ports the following examples to Camel K:
- https://camel.apache.org/manual/rest-dsl.html#_openapi_swagger_api
- https://github.com/apache/camel-examples/tree/camel-3.11.x/examples/openapi-cdi

Target of this setup was to make the local development experience better,
by providing a way to use not only inner classes in route file (`OpenApiRoute.java`) and avoid using
Maven or jitpack, which add major round trips to the development cycle.

## Solution
### Build a JAR
You can build the JAR file by `mvn install`.  
Be aware that you have to exclude the route file (`OpenApiRoute.java`) in the `pom.xml` as this file will already be provided over the
`kamel run ...` CLI command.

### Include the JAR
Now that you have the source code packed in the jar file you have to provide this jar file to the Camel K container,
where your Camel K route is running.  
You can do this by adding the following (modelines)[https://camel.apache.org/camel-k/1.6.x/cli/modeline.html]
to your route file (`OpenApiRoute.java`).
```
// camel-k: resource=file:../../../../../target/pojo-jar-1.0.0.jar
// camel-k: trait=jvm.classpath=/etc/camel/resources/pojo-jar-1.0.0.jar
```
It's important to add the JAR file as resource to be compressed.

### Run this route
To run this integration use the following command:
```bash
kamel run ./src/main/org/apache/camel/OpenApiRoute.java
```

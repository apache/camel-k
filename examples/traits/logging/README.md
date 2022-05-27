# Logging Trait

This type of trait tells us what is going on in a running integration pod. It does this by printing logs out to the standard output.

Logs are enabled by default on an integration. Just run this example:

     $ kamel run Logging.java

     $ kamel logs Logging

Or you can instead get the pod name and print its logs using kubectl: 

     $ kubectl get pods

     $ kubectl logs <pod name>

You can configure the log output with the `logging traits` flag:
      
Property | Type | Description 
---|---|---  
logging.enabled | boolean| We can set to false to disable logging
logging.color | boolean| makes logging output colorful. This makes skimming through the logs easier.
logging.json | boolean | Makes logs output to be in json format. We can use tools like `jq` to manipulate output.
logging.json-pretty-print | boolean | It's like using an in built `jq` to print our json output.
logging.level | string | This is just a verbosity level settings. We can just use `info`

for more traits options see this [Link.](https://camel.apache.org/camel-k/next/traits/logging.html).

manual log setting example using  `logging trait`:
     
     $ kamel run ./Logging.java --trait logging.enabled=true --trait logging.json=true \
     --trait logging.level=info

**Output**

The output of this result would be:

```
[2] {"timestamp":"2022-04-24T22:50:06.303Z","sequence":183,"loggerClassName":"org.slf4j.impl.Slf4jLogger","loggerName":"org.apache.camel.impl.engine.AbstractCamelContext","level":"INFO","message":"Apache Camel 3.14.1 (camel-1) started in 352ms (build:0ms init:293ms start:59ms)","threadName":"main","threadId":1,"mdc":{},"ndc":"","hostName":"rest-dsl-56668cc6dc-9zz7r","processName":"io.quarkus.bootstrap.runner.QuarkusEntryPoint","processId":1}
[2] {"timestamp":"2022-04-24T22:50:06.835Z","sequence":184,"loggerClassName":"org.jboss.logging.Logger","loggerName":"io.quarkus","level":"INFO","message":"camel-k-integration 1.8.2 on JVM (powered by Quarkus 2.7.0.Final) started in 14.348s. Listening on: http://0.0.0.0:8080","threadName":"main","threadId":1,"mdc":{},"ndc":"","hostName":"rest-dsl-56668cc6dc-9zz7r","processName":"io.quarkus.bootstrap.runner.QuarkusEntryPoint","processId":1}
[2] {"timestamp":"2022-04-24T22:50:06.842Z","sequence":185,"loggerClassName":"org.jboss.logging.Logger","loggerName":"io.quarkus","level":"INFO","message":"Profile prod activated. ","threadName":"main","threadId":1,"mdc":{},"ndc":"","hostName":"rest-dsl-56668cc6dc-9zz7r","processName":"io.quarkus.bootstrap.runner.QuarkusEntryPoint","processId":1}
[2] {"timestamp":"2022-04-24T22:50:06.842Z","sequence":186,"loggerClassName":"org.jboss.logging.Logger","loggerName":"io.quarkus","level":"INFO","message":"Installed features: [camel-attachments, camel-bean, camel-core, camel-direct, camel-java-joor-dsl, camel-k-core, camel-k-runtime, camel-platform-http, camel-rest, cdi, smallrye-context-propagation, vertx]","threadName":"main","threadId":1,"mdc":{},"ndc":"","hostName":"rest-dsl-56668cc6dc-9zz7r","processName":"io.quarkus.bootstrap.runner.QuarkusEntryPoint","processId":1}

```
- Logging would be enabled, and it's output would be in josn. But, there would be no colors for easy skimming.
- You would need to use your own jq to pretty print and parse the json output. 
## Using modeline 
An example of using a `modeline` to set the `logging traits` : 

     $ kamel run ./LoggingModeline.java 


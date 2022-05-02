# Examples showing how to use Camel K to connect to an IBM MQ Server

* Deploy the IBM MQ Server, as describe in the ibm-mq-server-deploy/README.md

* Change the IBM MQ Server address in the MQRoute.java class file

```
ibmip=`kubectl get svc/ibm-mq-server -ojsonpath="{.spec.clusterIP}"`; sed -i "/mqHost/s/\".*\"/\"$ibmip\"/g" MQRoute.java
```

For licensing reasons, the IBM MQ Java libraries are not defined in the routes themselves, but you can declare the dependency while running the integration. Alternatively you can use Kamel modeline to add the dependency in the route file as a header.

* Run the MQRoute.java. It is a producer and consumer of messages.

```
kamel run --dev MQRoute.java -d mvn:com.ibm.mq:com.ibm.mq.allclient:9.2.5.0
```

It will print the following output in the console

```
JmsConsumer[DEV.QUEUE.1]) Exchange[ExchangePattern: InOnly, BodyType: String, Body: Hello Camel K! #2]
```

* If you want to have a more streamlined declarative approach to run the integration using Kamelets, you can use the routes in YAML format.


The following kamel commands, has three distincts configurations:
1. Declare de dependency of IBM MQ Java library as previously mentioned.
2. Use the IBM MQ Server password set in a kubernetes `Secret` object.
3. Set the IBM MQ Server IP address as a property to run the integration, so the IBM MQ Cient can connect to the server.


Run the integration to generate messages and send them to the IBM MQ Queue (there is no output in the console)
```
kamel run --dev jms-ibm-mq-sink-binding.yaml -d mvn:com.ibm.mq:com.ibm.mq.allclient:9.2.5.0 --config secret:ibm-mq/ibm-mq-password -p serverName=`kubectl get svc/ibm-mq-server -ojsonpath="{.spec.clusterIP}"`
```

Run the integration to retrieve messages from the IBM MQ Queue and print in the console.
```
kamel run --dev jms-ibm-mq-source-binding.yaml -d mvn:com.ibm.mq:com.ibm.mq.allclient:9.2.5.0 --config secret:ibm-mq/ibm-mq-password -p serverName=`kubectl get svc/ibm-mq-server -ojsonpath="{.spec.clusterIP}"`
```

# Open API Camel K examples

Find useful examples about how to expose an Open API specification in a Camel K integration.

## Greetings example

Deploy the examples running

```
kamel run --dev --name greetings --open-api greetings-api.json greetings.groovy
```

Then you can test by calling the hello endpoint, ie:

```
$ curl -i http://192.168.49.2:31373/camel/greetings/hello
HTTP/1.1 200 OK
Accept: */*
name: hello
User-Agent: curl/7.68.0
transfer-encoding: chunked

Hello from hello
```
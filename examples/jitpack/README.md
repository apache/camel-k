# Jitpack Camel K examples

Find useful examples about how to use Jitpack in a Camel K integration.

## How to package a dependency

Camel K supports [Jitpack](https://jitpack.io/) to allow packaging your local code into a running Integration. The operator will recognize the major `git` repositories and provide all the needed configuration to package the dependency and locally install into Camel K artifacts repository. As an example, we can use [a sample jitpack project](https://github.com/squakez/samplejp), which contains some developments on `main` branch, a final release tagged as `v1.0` and other developments ongoing on `1.0.0` branch.

Within your route, you can import the code as you'd normally do with any other library:

```
import org.apache.camel.builder.RouteBuilder;
import acme.App;

public class Jitpack extends RouteBuilder {
  @Override
  public void configure() throws Exception {
      from("timer:tick?period=2000")
        .setBody()
          .simple(App.capitalize("hello"))
        .to("log:info");

  }
}
```

Once done, you must just reference the project in `kamel run -d` option, ie `-d github:squakez/samplejp`.

### Package the default branch (main)

You can choose to use the default dependency without specifying a tag or branch, that will fetch the source code on `main` branch:
```
kamel run Jitpack.java --dev -d github:squakez/samplejp

...
[1] 2021-11-23 16:00:59,305 INFO  [info] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: HELLO]
...
```

### Package another branch (1.0.0)

You can choose to compile the source code stored on a given branch, ie, on `1.0.0` branch:
```
kamel run Jitpack.java --dev -d github:squakez/samplejp:1.0.0-SNAPSHOT

...
[1] 2021-11-23 16:04:30,840 INFO  [info] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: v1.0.0-SNAPSHOT:HELLO]
...
```
### Package a fixed release tagged

You can also choose to package the source code released with a `tag`, ie `v1.0`:
```
kamel run Jitpack.java --dev -d github:squakez/samplejp:v1.0

...
[1] 2021-11-23 16:01:49,409 INFO  [info] (Camel (camel-1) thread #0 - timer://tick) Exchange[ExchangePattern: InOnly, BodyType: String, Body: v1.0.0:HELLO]
...
```


// ********************
//
// setup
//
// ********************

//val builder = bindings["builder"] as org.apache.camel.builder.RouteBuilder
//fun from(uri: String): org.apache.camel.model.RouteDefinition = builder.from(uri)

// ********************
//
// routes
//
// ********************

from("timer:tick")
    .process().message {
        m -> m.headers["MyHeader"] = "MyHeaderValue"
    }
    .to("log:info")
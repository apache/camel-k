
context {
    registry {
        bind("my-entry", "myRegistryEntry1")
        bind("my-proc", processor {
            e -> e.getIn().body = "Hello"
        })
    }
}


from("timer:tick")
    .process().message {
        m -> m.headers["MyHeader"] = "MyHeaderValue"
    }
    .to("log:info")
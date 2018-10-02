
restConfiguration {
    host = "my-host"
    port = "9192"
}

restConfiguration("undertow") {
    host = "my-undertow-host"
    port = "9193"
}


from("timer:tick")
    .process().message {
        m -> m.headers["MyHeader"] = "MyHeaderValue"
    }
    .to("log:info")

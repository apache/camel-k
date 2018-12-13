
from("timer:tick")
    .process().message {
        m -> m.headers["MyHeader"] = "MyHeaderValue"
    }
    .to("log:info")
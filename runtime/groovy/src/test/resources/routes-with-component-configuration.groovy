
component('seda') {
    // set value as method
    queueSize 1234

    // set value as property
    concurrentConsumers = 12
}

component('log') {
    formatter {
        'body ==> ' + in.body
    }
}


from('timer:tick')
    .to('log:info')

context {
    components {
        'seda' {
            // set value as method
            queueSize 1234

            // set value as property
            concurrentConsumers = 12
        }

        'log' {
            formatter {
                'body ==> ' + it.in.body
            }
        }
    }
}


from('timer:tick')
    .to('log:info')

context {
    components {
        'seda' {
            // set value as method
            queueSize 1234

            // set value as property
            concurrentConsumers = 12
        }

        'log' {
            exchangeFormatter = {
                'body ==> ' + it.in.body
            } as org.apache.camel.spi.ExchangeFormatter
        }
    }
}


from('timer:tick')
    .to('log:info')
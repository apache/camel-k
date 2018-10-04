import org.apache.camel.component.seda.SedaComponent

context {
    components {
        seda {
            // set value as method
            queueSize 1234

            // set value as property
            concurrentConsumers = 12
        }

        mySeda(SedaComponent) {
            // set value as method
            queueSize 4321

            // set value as property
            concurrentConsumers = 21
        }

        log {
            formatter {
                'body ==> ' + it.in.body
            }
        }
    }
}


from('timer:tick')
    .to('log:info')
import org.apache.camel.Exchange
import org.apache.camel.component.log.LogComponent
import org.apache.camel.component.seda.SedaComponent

context {

    components {

        component<LogComponent>("log") {
            setExchangeFormatter {
                e: Exchange -> "" + e.getIn().body
            }
        }

        component<SedaComponent>("seda") {
            queueSize = 1234
            concurrentConsumers = 12
        }

        component<SedaComponent>("mySeda") {
            queueSize = 4321
            concurrentConsumers = 21
        }
    }
}

from("timer:tick")
    .to("log:info")
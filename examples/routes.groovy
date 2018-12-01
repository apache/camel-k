import java.util.concurrent.ThreadLocalRandom

//
// To run this integrations use:
//
//     kamel run --runtime groovy examples/routes.groovy
//
// Or leveraging runtime detection
//
//     kamel run examples/routes.groovy
//

context {

    //
    // configure components
    //
    components {
        log {
            formatter {
                'body: ' + it.in.body + ', random-value: ' + it.in.headers['RandomValue']
            }
        }
    }

    //
    // configure registry
    //
    registry {
        myProcessor = processor {
            it.in.headers['RandomValue'] = ThreadLocalRandom.current().nextInt()
        }
    }
}


from('timer:groovy?period=1s')
    .routeId('groovy')
    .setBody()
        .constant('Hello Camel K!')
    .process('myProcessor')
    .to('log:info')
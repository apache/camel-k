//
// To run this integrations use:
//
//     kamel run -d camel:groovy runtime/examples/routes.groovy
//

rnd = new Random()

from('timer:groovy?period=1s')
    .routeId('groovy')
    .setBody()
        .constant('Hello Camel K!')
    .process {
        it.in.headers['RandomValue'] = rnd.nextInt()
    }
    .to('log:info?showHeaders=true')
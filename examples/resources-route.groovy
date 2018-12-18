//
// To run this integrations use:
//
//     kamel run --resource examples/resources-data.txt examples/resources-route.groovy
//

from('timer:resources')
    .routeId('resources')
    .setBody()
        .simple("resource:platform:resources-data.txt")
    .log('file content is: ${body}')

//
//  kamel run --dev --name greetings --dependency camel-undertow --property camel.rest.port=8080 --open-api examples/greetings-api.json --logging-level org.apache.camel.k=DEBUG examples/greetings.groovy 
// 

from('direct:greeting-api')
    .to('log:api?showAll=true&multiline=true') 
    .setBody()
        .simple('Hello from ${headers.name}')

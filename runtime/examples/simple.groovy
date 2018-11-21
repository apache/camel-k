
from('timer:groovy?period=1s')
    .routeId('groovy')
    .setBody()
        .simple('Hello Camel K from ${routeId}')
    .to('log:info?showAll=false')

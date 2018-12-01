
from('timer:js?period=1s')
    .routeId('js')
    .setBody()
        .simple('Hello Camel K from ${routeId}')
    .to('log:info?multiline=true')
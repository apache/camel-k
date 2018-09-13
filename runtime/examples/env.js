//
// To run this integrations use:
//
//     kamel run -e MY_MESSAGE=test-env runtime/examples/env.js
//

from('timer:env?period=1s')
    .routeId('env')
    .log('{{env:MY_MESSAGE}}')
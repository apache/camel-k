//
// To run this integrations use:
//
//     kamel run -p my.message=test-props examples/props.js
//

from('timer:props?period=1s')
    .routeId('props')
    .log('{{my.message}}')
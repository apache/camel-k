//
// To run this integrations use:
//
//     kamel run -d camel:dns examples/dns.js
//
// Or simply (since dependency auto-detection is enabled by default):
//
//     kamel run examples/dns.js
//

from('timer:dns?period=1s')
    .routeId('dns')
    .setHeader('dns.domain')
        .constant('www.google.com')
    .to('dns:ip')
    .to('log:dns');

// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//
// To run this integrations use:
//
//     kamel run --name=withrest --dependency=camel-undertow examples/routes-rest.js
//

// TODO: disabled because of https://github.com/oracle/graal/issues/1247
// l = components.get('log');
// l.exchangeFormatter = function(e) {
//     return "log - body=" + e.in.body + ", headers=" + e.in.headers
// };

c = restConfiguration();
c.setComponent('undertow');
c.setPort('8080');

// TODO: disabled because of https://github.com/oracle/graal/issues/1247
// function proc(e) {
//     e.getIn().setHeader('RandomValue', Math.floor((Math.random() * 100) + 1))
// }

rest('/say/hello')
    .produces("text/plain")
    .get().route()
        .transform().constant("Hello World");

from('timer:js?period=1s')
    .routeId('js')
    .setBody()
        .constant('Hello Camel K')
    .to('log:info')

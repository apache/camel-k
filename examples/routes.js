// camel-k: language=js
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
//     kamel run examples/routes.js
//
const org_apache_camel_Processor = Java.type("org.apache.camel.Processor");
const Processor = Java.extend(org_apache_camel_Processor);

l = components.get('log');
l.setExchangeFormatter(e => {
    return "body=" + e.getIn().getBody() + ", headers=" + e.getIn().getHeaders()
})

from('timer:js?period=1000')
    .routeId('js')
    .setBody()
        .simple('Hello Camel K')
    .process(new Processor(e => {
        e.getIn().setHeader('RandomValue', Math.floor((Math.random() * 100) + 1))
    }))
    .to('log:info');

// camel-k: language=groovy
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//
// To run this integration use:
//
//     kamel run --name pulsar-groovy --dev -d camel-pulsar examples/pulsar.groovy
//
//  Notes: 
//  camel-pulsar may be unecessary as camel-k can detect automatically dependencies from component calls for example from("pulsar://localhost:6650/tenant/namespace/topic")
//  --dev mode is usual to debug and test stuff
//  null pointer exception is possible if the PulsarClient is not properly cofigured


beans {
    //let's assume pulsar is deployed for testing inside docker in local machine
    pulsarClient = org.apache.pulsar.client.api.PulsarClient.builder().serviceUrl("pulsar://host.docker.internal:6650").build() // creates a new bean which will be detected by camel and set as primary PulsarClient
}

camel {
    dataFormats { 
        jackson(org.apache.camel.component.jackson.JacksonDataFormat) { 
            include = "NON_NULL" // assuming message format will be in JSON this will configure it to ignore fields like "somefield": null in case those are sent
        }
    }
}

onException(Exception.class).to("log:error") // doing something with errors - with no extra configuration this will push it to console

from("pulsar:non-persistent://public/default/testtopic") // a call to pulsar component telling it to use non persistent messaging (are not stored to anywhere) in default namespace listening to topic testtopic
    //assumig UTF should be used, parse bytes to UTF-8 then to json which could possibly avoid special character represented porrly due to default local machine encoding being used instead of UTF-8
    .convertBodyTo(String.class, "UTF-8").unmarshal().json(org.apache.camel.model.dataformat.JsonLibrary.Jackson) // simple json parsing
    .process {
        def someMessage = it.in.body["message"] // json message is no available as hashmap
        
        if (it.in.body["error"]) {
            it.out.body = [error: "Testing error message", details: someMessage, thisShouldBeIgnored: null]
        } else if (it.in.body["exception"]) {
            throw new Exception("I'm logging")
        } else {
            it.out.body = [success: true, message: someMessage, thisShouldBeIgnored: null]
        }
    }
    .choice() // switch based on message content
        //marshal after message has been parsed with custom marshaller ignoring null field
       .when(simple('${body["error"]}')).marshal('jackson').to("pulsar:non-persistent://public/default/errors") // log + pass error to pulsar
       .otherwise().marshal('jackson').to('log:info').to("pulsar:non-persistent://public/default/receivedtopic") // publish message to other topic

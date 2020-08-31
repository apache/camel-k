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
// To run this integrations use:
//
//     kamel run -e BROKER_URL=event-bus-amqp-0-svc.messaging.svc.cluster.local -d camel-amqp examples/amqpConsumer.java
//
import org.apache.camel.BindToRegistry;

public class amqpConsume extends org.apache.camel.builder.RouteBuilder {

    @BindToRegistry
    public javax.jms.ConnectionFactory connectionFactory() {
        return new org.apache.qpid.jms.JmsConnectionFactory("amqp://" + java.lang.System.getenv("BROKER_URL"));
    }

    @Override
    public void configure() throws Exception {

        from("amqp:topic:example?exchangePattern=InOnly")
            .to("log:info");

    }
}

/**
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
package org.apache.camel.component.knative;

import java.util.Arrays;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.impl.SimpleRegistry;

import static org.apache.camel.util.CollectionHelper.mapOf;

public class KnativeComponentMain {
    public static void main(String[] args) throws Exception {
        KnativeComponent component = new KnativeComponent();
        component.setEnvironment(newEnv());

        SimpleRegistry registry = new SimpleRegistry();
        registry.put("knative", component);

        DefaultCamelContext context = new DefaultCamelContext(registry);

        try {
            context.disableJMX();
            context.addRoutes(new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    from("knative:endpoint/ep1")
                        .convertBodyTo(String.class)
                        .to("log:ep1?showAll=true&multiline=true")
                        .setBody().constant("Hello from CH1");
                    from("knative:endpoint/ep2")
                        .convertBodyTo(String.class)
                        .to("log:ep2?showAll=true&multiline=true")
                        .setBody().constant("Hello from CH2");
                }
            });

            context.start();

            Thread.sleep(Integer.MAX_VALUE);
        } finally {
            context.stop();
        }
    }

    private static KnativeEnvironment newEnv() {
        return new KnativeEnvironment(Arrays.asList(
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "ep1",
                "localhost",
                8080,
                mapOf(
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain",
                    Knative.FILTER_HEADER_NAME, "CE-Source",
                    Knative.FILTER_HEADER_VALUE, "CE1"
                )),
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "ep2",
                "localhost",
                8080,
                mapOf(
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain",
                    Knative.FILTER_HEADER_NAME, "CE-Source",
                    Knative.FILTER_HEADER_VALUE, "CE2"
                ))
        ));
    }
}

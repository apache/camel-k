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

import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.util.Arrays;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.camel.CamelContext;
import org.apache.camel.Exchange;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.cloud.ServiceDefinition;
import org.apache.camel.component.knative.ce.CloudEventsProcessors;
import org.apache.camel.component.mock.MockEndpoint;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.test.AvailablePortFinder;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import static org.apache.camel.util.CollectionHelper.mapOf;

public class CloudEventsV01Test {

    private CamelContext context;

    // **************************
    //
    // Setup
    //
    // **************************

    @BeforeEach
    public void before() {
        this.context = new DefaultCamelContext();
    }

    @AfterEach
    public void after() throws Exception {
        if (this.context != null) {
            this.context.stop();
        }
    }

    // **************************
    //
    // Tests
    //
    // **************************

    @Test
    void testInvokeEndpoint() throws Exception {
        final int port = AvailablePortFinder.getNextAvailable();

        KnativeEnvironment env = new KnativeEnvironment(Arrays.asList(
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "myEndpoint",
                "localhost",
                port,
                mapOf(
                    ServiceDefinition.SERVICE_META_PATH, "/a/path",
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain"
                ))
        ));

        KnativeComponent component = context.getComponent("knative", KnativeComponent.class);
        component.setCloudEventsSpecVersion(CloudEventsProcessors.v01.getVersion());
        component.setEnvironment(env);

        context.addRoutes(new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                from("direct:source")
                    .to("knative:endpoint/myEndpoint");

                fromF("netty4-http:http://localhost:%d/a/path", port)
                    .to("mock:ce");
            }
        });

        context.start();

        MockEndpoint mock = context.getEndpoint("mock:ce", MockEndpoint.class);
        mock.expectedMessageCount(1);
        mock.expectedHeaderReceived("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
        mock.expectedHeaderReceived("CE-EventType", "org.apache.camel.event");
        mock.expectedHeaderReceived("CE-Source", "knative://endpoint/myEndpoint");
        mock.expectedHeaderReceived(Exchange.CONTENT_TYPE, "text/plain");
        mock.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventTime"));
        mock.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventID"));
        mock.expectedBodiesReceived("test");

        context.createProducerTemplate().send(
            "direct:source",
            e -> {
                e.getIn().setBody("test");
            }
        );

        mock.assertIsSatisfied();
    }

    @Test
    void testConsumeStructuredContent() throws Exception {
        final int port = AvailablePortFinder.getNextAvailable();

        KnativeEnvironment env = new KnativeEnvironment(Arrays.asList(
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "myEndpoint",
                "localhost",
                port,
                mapOf(
                    ServiceDefinition.SERVICE_META_PATH, "/a/path",
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain"
                ))
        ));

        KnativeComponent component = context.getComponent("knative", KnativeComponent.class);
        component.setCloudEventsSpecVersion(CloudEventsProcessors.v01.getVersion());
        component.setEnvironment(env);

        context.addRoutes(new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                from("knative:endpoint/myEndpoint")
                    .to("mock:ce");

                from("direct:source")
                    .toF("netty4-http:http://localhost:%d/a/path", port);
            }
        });

        context.start();

        MockEndpoint mock = context.getEndpoint("mock:ce", MockEndpoint.class);
        mock.expectedMessageCount(1);
        mock.expectedHeaderReceived("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
        mock.expectedHeaderReceived("CE-EventType", "org.apache.camel.event");
        mock.expectedHeaderReceived("CE-EventID", "myEventID");
        mock.expectedHeaderReceived("CE-Source", "/somewhere");
        mock.expectedHeaderReceived(Exchange.CONTENT_TYPE, Knative.MIME_STRUCTURED_CONTENT_MODE);
        mock.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventTime"));
        mock.expectedBodiesReceived("test");

        context.createProducerTemplate().send(
            "direct:source",
            e -> {
                e.getIn().setHeader(Exchange.CONTENT_TYPE, Knative.MIME_STRUCTURED_CONTENT_MODE);
                e.getIn().setBody(new ObjectMapper().writeValueAsString(mapOf(
                    "cloudEventsVersion", CloudEventsProcessors.v01.getVersion(),
                    "eventType", "org.apache.camel.event",
                    "eventID", "myEventID",
                    "eventTime", DateTimeFormatter.ISO_OFFSET_DATE_TIME.format(ZonedDateTime.now()),
                    "source", "/somewhere",
                    "data", "test"
                )));
            }
        );

        mock.assertIsSatisfied();
    }

    @Test
    void testConsumeContent() throws Exception {
        final int port = AvailablePortFinder.getNextAvailable();

        KnativeEnvironment env = new KnativeEnvironment(Arrays.asList(
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "myEndpoint",
                "localhost",
                port,
                mapOf(
                    ServiceDefinition.SERVICE_META_PATH, "/a/path",
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain"
                ))
        ));

        KnativeComponent component = context.getComponent("knative", KnativeComponent.class);
        component.setCloudEventsSpecVersion(CloudEventsProcessors.v01.getVersion());
        component.setEnvironment(env);

        context.addRoutes(new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                from("knative:endpoint/myEndpoint")
                    .to("mock:ce");

                from("direct:source")
                    .toF("http4://localhost:%d/a/path", port);
            }
        });

        context.start();

        MockEndpoint mock = context.getEndpoint("mock:ce", MockEndpoint.class);
        mock.expectedMessageCount(1);
        mock.expectedHeaderReceived("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
        mock.expectedHeaderReceived("CE-EventType", "org.apache.camel.event");
        mock.expectedHeaderReceived("CE-EventID", "myEventID");
        mock.expectedHeaderReceived("CE-Source", "/somewhere");
        mock.expectedHeaderReceived(Exchange.CONTENT_TYPE, "text/plain");
        mock.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventTime"));
        mock.expectedBodiesReceived("test");

        context.createProducerTemplate().send(
            "direct:source",
            e -> {
                e.getIn().setHeader(Exchange.CONTENT_TYPE, "text/plain");
                e.getIn().setHeader("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
                e.getIn().setHeader("CE-EventType", "org.apache.camel.event");
                e.getIn().setHeader("CE-EventID", "myEventID");
                e.getIn().setHeader("CE-EventTime", DateTimeFormatter.ISO_OFFSET_DATE_TIME.format(ZonedDateTime.now()));
                e.getIn().setHeader("CE-Source", "/somewhere");
                e.getIn().setBody("test");
            }
        );

        mock.assertIsSatisfied();
    }

    @Test
    void testConsumeContentWithFilter() throws Exception {
        final int port = AvailablePortFinder.getNextAvailable();

        KnativeEnvironment env = new KnativeEnvironment(Arrays.asList(
            new KnativeEnvironment.KnativeServiceDefinition(
                Knative.Type.endpoint,
                Knative.Protocol.http,
                "ep1",
                "localhost",
                port,
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
                port,
                mapOf(
                    Knative.KNATIVE_EVENT_TYPE, "org.apache.camel.event",
                    Knative.CONTENT_TYPE, "text/plain",
                    Knative.FILTER_HEADER_NAME, "CE-Source",
                    Knative.FILTER_HEADER_VALUE, "CE2"
                ))
        ));

        KnativeComponent component = context.getComponent("knative", KnativeComponent.class);
        component.setCloudEventsSpecVersion(CloudEventsProcessors.v01.getVersion());
        component.setEnvironment(env);

        context.addRoutes(new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                from("knative:endpoint/ep1")
                    .convertBodyTo(String.class)
                    .to("log:ce1?showAll=true&multiline=true")
                    .to("mock:ce1");
                from("knative:endpoint/ep2")
                    .convertBodyTo(String.class)
                    .to("log:ce2?showAll=true&multiline=true")
                    .to("mock:ce2");

                from("direct:source")
                    .setBody()
                        .constant("test")
                    .setHeader(Exchange.HTTP_METHOD)
                        .constant("POST")
                    .setHeader(Exchange.HTTP_QUERY)
                        .simple("filter.headerName=CE-Source&filter.headerValue=${header.FilterVal}")
                    .toD("http4://localhost:" + port);
            }
        });

        context.start();

        MockEndpoint mock1 = context.getEndpoint("mock:ce1", MockEndpoint.class);
        mock1.expectedMessageCount(1);
        mock1.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventTime"));
        mock1.expectedHeaderReceived("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
        mock1.expectedHeaderReceived("CE-EventType", "org.apache.camel.event");
        mock1.expectedHeaderReceived("CE-EventID", "myEventID1");
        mock1.expectedHeaderReceived("CE-Source", "CE1");
        mock1.expectedBodiesReceived("test");

        MockEndpoint mock2 = context.getEndpoint("mock:ce2", MockEndpoint.class);
        mock2.expectedMessageCount(1);
        mock2.expectedMessagesMatches(e -> e.getIn().getHeaders().containsKey("CE-EventTime"));
        mock2.expectedHeaderReceived("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
        mock2.expectedHeaderReceived("CE-EventType", "org.apache.camel.event");
        mock2.expectedHeaderReceived("CE-EventID", "myEventID2");
        mock2.expectedHeaderReceived("CE-Source", "CE2");
        mock2.expectedBodiesReceived("test");

        context.createProducerTemplate().send(
            "direct:source",
            e -> {
                e.getIn().setHeader("FilterVal", "CE1");
                e.getIn().setHeader("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
                e.getIn().setHeader("CE-EventType", "org.apache.camel.event");
                e.getIn().setHeader("CE-EventID", "myEventID1");
                e.getIn().setHeader("CE-EventTime", DateTimeFormatter.ISO_OFFSET_DATE_TIME.format(ZonedDateTime.now()));
                e.getIn().setHeader("CE-Source", "CE1");
            }
        );
        context.createProducerTemplate().send(
            "direct:source",
            e -> {
                e.getIn().setHeader("FilterVal", "CE2");
                e.getIn().setHeader("CE-CloudEventsVersion", CloudEventsProcessors.v01.getVersion());
                e.getIn().setHeader("CE-EventType", "org.apache.camel.event");
                e.getIn().setHeader("CE-EventID", "myEventID2");
                e.getIn().setHeader("CE-EventTime", DateTimeFormatter.ISO_OFFSET_DATE_TIME.format(ZonedDateTime.now()));
                e.getIn().setHeader("CE-Source", "CE2");
            }
        );

        mock1.assertIsSatisfied();
        mock2.assertIsSatisfied();
    }
}

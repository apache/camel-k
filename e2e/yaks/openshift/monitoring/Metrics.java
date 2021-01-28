// camel-k: language=java trait=prometheus.enabled=true

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

import org.apache.camel.Exchange;
import org.apache.camel.LoggingLevel;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.microprofile.metrics.MicroProfileMetricsConstants;

import javax.enterprise.context.ApplicationScoped;

/**
 * This example registers the following metrics:
 * <ul>
 *     <li>{@code camel-k-example-metrics-attempt}</code>: meters the number of calls
 *     made to the service to process incoming events</li>
 *     <li>{@code camel-k-example-metrics-error}</code>: meters the number of errors
 *     corresponding to the number of events that haven't been processed</li>
 *     <li>{@code camel-k-example-metrics-generated}</code>: meters the number of events to be processed</li>
 *     <li>{@code camel-k-example-metrics-redelivery}</code>: meters the number of retries
 *     made to process the events</li>
 *     <li>{@code camel-k-example-metrics-success}</code>: meters the number of events successfully processed</li>
 * </ul>
 * The invariant being: {@code attempt = redelivery - success - error}.
 * <p> In addition, a ratio gauge {@code success-ratio = success / generated} is registered.
 *
 */
@ApplicationScoped
public class Metrics extends RouteBuilder {

    @Override
    public void configure() {
        onException()
            .handled(true)
            .maximumRedeliveries(2)
            .logStackTrace(false)
            .logExhausted(false)
            .log(LoggingLevel.ERROR, "Failed processing ${body}")
            .to("microprofile-metrics:meter:camel-k-example-metrics-redelivery?mark=2")
            // The 'error' meter
            .to("microprofile-metrics:meter:camel-k-example-metrics-error");

        from("timer:stream?period=1000")
            .routeId("unreliable-service")
            .setBody(header(Exchange.TIMER_COUNTER).prepend("event #"))
            .log("Processing ${body}...")
            // The 'generated' meter
            .to("microprofile-metrics:meter:camel-k-example-metrics-generated")
            // TODO: replace with lookup by type as soon as CAMEL-15217 gets fixed
            // The 'attempt' meter via @Metered interceptor
            .bean("service")
            .filter(header(Exchange.REDELIVERED))
                .log(LoggingLevel.WARN, "Processed ${body} after ${header.CamelRedeliveryCounter} retries")
                .setHeader(MicroProfileMetricsConstants.HEADER_METER_MARK, header(Exchange.REDELIVERY_COUNTER))
                // The 'redelivery' meter
                .to("microprofile-metrics:meter:camel-k-example-metrics-redelivery")
            .end()
            .log("Successfully processed ${body}")
            // The 'success' meter
            .to("microprofile-metrics:meter:camel-k-example-metrics-success");
    }
}

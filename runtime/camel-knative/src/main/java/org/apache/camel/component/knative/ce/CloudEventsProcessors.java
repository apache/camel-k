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
package org.apache.camel.component.knative.ce;

import java.util.function.Function;

import org.apache.camel.Processor;
import org.apache.camel.component.knative.KnativeEndpoint;

public enum CloudEventsProcessors {
    v01("0.1", V01.PRODUCER, V01.CONSUMER),
    v02("0.2", V02.PRODUCER, V02.CONSUMER);

    private final String version;
    private final Function<KnativeEndpoint, Processor> producer;
    private final Function<KnativeEndpoint, Processor> consumer;

    CloudEventsProcessors(String version, Function<KnativeEndpoint, Processor> producer, Function<KnativeEndpoint, Processor> consumer) {
        this.version = version;
        this.producer = producer;
        this.consumer = consumer;
    }

    public String getVersion() {
        return version;
    }

    public Processor producerProcessor(KnativeEndpoint endpoint) {
        return this.producer.apply(endpoint);
    }

    public Processor consumerProcessor(KnativeEndpoint endpoint) {
        return this.consumer.apply(endpoint);
    }

    // **************************
    //
    // Helpers
    //
    // **************************

    public static CloudEventsProcessors forSpecversion(String version) {
        for (CloudEventsProcessors ce : CloudEventsProcessors.values()) {
            if (ce.version.equals(version)) {
                return ce;
            }
        }

        throw new IllegalArgumentException("Unable to find processors for spec version: " +  version);
    }
}

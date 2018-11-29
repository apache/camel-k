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

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jdk8.Jdk8Module;

public final class Knative {
    public static final ObjectMapper MAPPER = new ObjectMapper().registerModule(new Jdk8Module());


    public static final String HTTP_COMPONENT = "knative-http";
    public static final String KNATIVE_PROTOCOL = "knative.protocol";
    public static final String KNATIVE_TYPE = "knative.type";
    public static final String KNATIVE_EVENT_TYPE = "knative.event.type";
    public static final String FILTER_HEADER_NAME = "filter.header.name";
    public static final String FILTER_HEADER_VALUE = "filter.header.value";
    public static final String CONTENT_TYPE = "content.type";
    public static final String MIME_STRUCTURED_CONTENT_MODE = "application/cloudevents+json";

    private Knative() {
    }

    public enum Type {
        endpoint,
        channel
    }

    public enum Protocol {
        http,
        https
    }
}

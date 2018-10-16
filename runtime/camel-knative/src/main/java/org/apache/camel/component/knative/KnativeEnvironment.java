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

import java.io.InputStream;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Stream;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;
import org.apache.camel.CamelContext;
import org.apache.camel.cloud.ServiceDefinition;
import org.apache.camel.impl.cloud.DefaultServiceDefinition;
import org.apache.camel.util.CollectionHelper;
import org.apache.camel.util.ResourceHelper;
import org.apache.camel.util.StringHelper;

/*
 * Assuming it is loaded from a json for now
 */
public class KnativeEnvironment {
    private final List<KnativeServiceDefinition> services;

    @JsonCreator
    public KnativeEnvironment(
        @JsonProperty(value = "services", required = true) List<KnativeServiceDefinition> services) {

        this.services = new ArrayList<>(services);
    }

    public Stream<KnativeServiceDefinition> stream() {
        return services.stream();
    }

    public Optional<KnativeServiceDefinition> lookupService(Knative.Type type, String name) {
        final String contextPath = StringHelper.after(name, "/");
        final String serviceName = (contextPath == null) ? name : StringHelper.before(name, "/");

        return services.stream()
            .filter(definition -> {
                return Objects.equals(type.name(), definition.getMetadata().get(Knative.KNATIVE_TYPE))
                    && Objects.equals(serviceName, definition.getName());
            })
            .map(definition -> {
                //
                // The context path set on the endpoint  overrides the one
                // eventually provided by the service definition.
                //
                if (contextPath != null) {
                    return new KnativeServiceDefinition(
                        definition.getType(),
                        definition.getProtocol(),
                        definition.getName(),
                        definition.getHost(),
                        definition.getPort(),
                        KnativeSupport.mergeMaps(
                            definition.getMetadata(),
                            Collections.singletonMap(ServiceDefinition.SERVICE_META_PATH, "/" + contextPath)
                        )
                    );
                }

                return definition;
            })
            .findFirst();
    }

    public KnativeServiceDefinition mandatoryLookupService(Knative.Type type, String name) {
        return lookupService(type, name).orElseThrow(
            () -> new IllegalArgumentException("Unable to find the service \"" + name + "\" with type \"" + type + "\"")
        );
    }

    // ************************
    //
    // Helpers
    //
    // ************************

    public static KnativeEnvironment mandatoryLoadFromResource(CamelContext context, String path) throws Exception {
        try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(context, path)) {

            //
            // read the knative environment from a file formatted as json, i.e. :
            //
            // {
            //     "services": [
            //         {
            //              "type": "channel|endpoint",
            //              "protocol": "http",
            //              "name": "",
            //              "host": "",
            //              "port": "",
            //              "metadata": {
            //                  "service.path": "",
            //                  "knative.event.type": ""
            //              }
            //         },
            //     ]
            // }
            //
            //
            return Knative.MAPPER.readValue(is, KnativeEnvironment.class);
        }
    }

    // ************************
    //
    // Types
    //
    // ************************

    public final static class KnativeServiceDefinition extends DefaultServiceDefinition {
        @JsonCreator
        public KnativeServiceDefinition(
            @JsonProperty(value = "type", required = true) Knative.Type type,
            @JsonProperty(value = "protocol", required = true) Knative.Protocol protocol,
            @JsonProperty(value = "name", required = true) String name,
            @JsonProperty(value = "host", required = true) String host,
            @JsonProperty(value = "port", required = true) int port,
            @JsonProperty(value = "metadata", required = false) Map<String, String> metadata) {

            super(
                UUID.randomUUID().toString(),
                name,
                host,
                port,
                KnativeSupport.mergeMaps(
                    metadata,
                    CollectionHelper.mapOf(
                        Knative.KNATIVE_TYPE, type.name(),
                        Knative.KNATIVE_PROTOCOL, protocol.name())
                )
            );
        }

        public Knative.Type getType() {
            return Knative.Type.valueOf(getMetadata().get(Knative.KNATIVE_TYPE));
        }

        public Knative.Protocol getProtocol() {
            return Knative.Protocol.valueOf(getMetadata().get(Knative.KNATIVE_PROTOCOL));
        }

        public String getPath() {
            return getMetadata().get(ServiceDefinition.SERVICE_META_PATH);
        }

        public String getEventType() {
            return getMetadata().get(Knative.KNATIVE_EVENT_TYPE);
        }
    }
}

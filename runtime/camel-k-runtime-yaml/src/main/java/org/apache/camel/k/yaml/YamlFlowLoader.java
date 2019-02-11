/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.apache.camel.k.yaml;

import java.io.InputStream;
import java.util.Collections;
import java.util.List;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.jsontype.NamedType;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.dataformat.yaml.YAMLGenerator;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.Source;
import org.apache.camel.k.support.URIResolver;
import org.apache.camel.k.yaml.model.Endpoint;
import org.apache.camel.k.yaml.model.Flow;
import org.apache.camel.k.yaml.model.Step;
import org.apache.camel.k.yaml.model.StepHandler;
import org.apache.camel.model.ProcessorDefinition;
import org.apache.camel.spi.FactoryFinder;

public class YamlFlowLoader implements RoutesLoader {
    private final ObjectMapper mapper;

    public YamlFlowLoader() {
        YAMLFactory yamlFactory = new YAMLFactory()
            .configure(YAMLGenerator.Feature.MINIMIZE_QUOTES, true)
            .configure(YAMLGenerator.Feature.ALWAYS_QUOTE_NUMBERS_AS_STRINGS, true)
            .configure(YAMLGenerator.Feature.USE_NATIVE_TYPE_ID, false);

        this.mapper = new ObjectMapper(yamlFactory)
            .setSerializationInclusion(JsonInclude.Include.NON_EMPTY)
            .enable(SerializationFeature.INDENT_OUTPUT);

        mapper.registerSubtypes(new NamedType(Endpoint.class, Endpoint.KIND));
    }

    @Override
    public List<String> getSupportedLanguages() {
        return Collections.singletonList("flow");
    }

    @SuppressWarnings("uncheked")
    @Override
    public RouteBuilder load(Runtime.Registry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                try (InputStream is = URIResolver.resolve(getContext(), source)) {
                    for (Flow flow: mapper.readValue(is, Flow[].class)) {
                        final List<Step> steps = flow.getSteps();
                        final int size = steps.size();
                        final FactoryFinder finder = getContext().getFactoryFinder(Step.RESOURCE_PATH);

                        ProcessorDefinition<?> definition = null;

                        for (int i = 0; i < size; i++) {
                            Step step = steps.get(i);

                            if (i == 0) {
                                // force the cast so it will fail at runtime
                                // if the step is not of the right type
                                definition = from(((Endpoint) step).getUri());

                                continue;
                            }

                            if (definition == null) {
                                throw new IllegalStateException("No route definition");
                            }

                            StepHandler<Step> handler = (StepHandler<Step>)finder.newInstance(step.getKind());
                            if (handler == null) {
                                throw new IllegalStateException("No handler for step with kind: " + step.getKind());
                            }

                            definition = handler.handle(step, definition);
                        }
                    }
                }
            }
        };
    }
}

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
package org.apache.camel.k.jvm.loader;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.List;
import java.util.Map;

import org.apache.camel.CamelContext;
import org.apache.camel.RoutesBuilder;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.Source;
import org.apache.camel.k.support.URIResolver;
import org.apache.camel.model.rest.RestConfigurationDefinition;
import org.apache.camel.spi.RestConfiguration;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.joor.Reflect;

public class JavaSourceLoader implements RoutesLoader {
    @Override
    public List<String> getSupportedLanguages() {
        return Collections.singletonList("java");
    }

    @Override
    public RouteBuilder load(Runtime.Registry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                final CamelContext context = getContext();

                try (InputStream is = URIResolver.resolve(context, source)) {
                    String name = source.getLocation();
                    name = StringUtils.substringAfter(name, ":");
                    name = StringUtils.removeEnd(name, ".java");

                    if (name.contains("/")) {
                        name = StringUtils.substringAfterLast(name, "/");
                    }

                    RoutesBuilder builder = Reflect.compile(name, IOUtils.toString(is, StandardCharsets.UTF_8)).create().get();

                    // Wrap routes builder
                    includeRoutes(builder);

                    if (builder instanceof RouteBuilder) {
                        Map<String, RestConfigurationDefinition> configurations = ((RouteBuilder) builder).getRestConfigurations();

                        //
                        // TODO: RouteBuilder.getRestConfigurations() should not
                        //       return null
                        //
                        if (configurations != null) {
                            for (RestConfigurationDefinition definition : configurations.values()) {
                                RestConfiguration conf = definition.asRestConfiguration(context);

                                //
                                // this is an hack to copy routes configuration
                                // to the camel context
                                //
                                // TODO: fix RouteBuilder.includeRoutes to include
                                //       rest configurations
                                //
                                context.addRestConfiguration(conf);
                            }
                        }
                    }
                }
            }
        };
    }
}

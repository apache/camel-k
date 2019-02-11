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
import java.util.Collections;
import java.util.List;
import javax.xml.bind.UnmarshalException;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.Source;
import org.apache.camel.k.support.URIResolver;
import org.apache.camel.model.RoutesDefinition;
import org.apache.camel.model.rest.RestsDefinition;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class XmlLoader implements RoutesLoader {
    private static final Logger LOGGER = LoggerFactory.getLogger(XmlLoader.class);

    @Override
    public List<String> getSupportedLanguages() {
        return Collections.singletonList("xml");
    }

    @Override
    public RouteBuilder load(Runtime.Registry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                try (InputStream is = URIResolver.resolve(getContext(), source)) {
                    try {
                        RoutesDefinition definition = getContext().loadRoutesDefinition(is);
                        LOGGER.debug("Loaded {} routes from {}", definition.getRoutes().size(), source);

                        setRouteCollection(definition);
                    } catch (IllegalArgumentException e) {
                        // ignore
                    } catch (UnmarshalException e) {
                        LOGGER.debug("Unable to load RoutesDefinition: {}", e.getMessage());
                    }
                }

                try (InputStream is = URIResolver.resolve(getContext(), source)) {
                    try {
                        RestsDefinition definition = getContext().loadRestsDefinition(is);
                        LOGGER.debug("Loaded {} rests from {}", definition.getRests().size(), source);

                        setRestCollection(definition);
                    } catch(IllegalArgumentException e) {
                        // ignore
                    } catch (UnmarshalException e) {
                        LOGGER.debug("Unable to load RestsDefinition: {}", e.getMessage());
                    }
                }
            }
        };
    }
}

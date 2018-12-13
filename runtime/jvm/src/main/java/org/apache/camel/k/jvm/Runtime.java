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
package org.apache.camel.k.jvm;

import java.util.Map;
import java.util.Properties;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

import org.apache.camel.CamelContext;
import org.apache.camel.ProducerTemplate;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.component.properties.PropertiesComponent;
import org.apache.camel.impl.CompositeRegistry;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.Source;
import org.apache.camel.main.MainSupport;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public final class Runtime extends MainSupport {
    private static final Logger LOGGER = LoggerFactory.getLogger(Runtime.class);

    private final ConcurrentMap<String, CamelContext> contextMap;
    private final RuntimeRegistry registry = new SimpleRuntimeRegistry();

    public Runtime() {
        this.contextMap = new ConcurrentHashMap<>();
    }

    public void load(String[] routes) throws Exception {
        for (String route: routes) {
            final Source source = Source.create(route);
            final RoutesLoader loader = RuntimeSupport.loaderFor(getCamelContext(), source);
            final RouteBuilder builder = loader.load(registry, source);

            if (builder == null) {
                throw new IllegalStateException("Unable to load route from: " + route);
            }

            LOGGER.info("Routes: {}", route);

            addRouteBuilder(builder);
        }
    }

    public RuntimeRegistry getRegistry() {
        return registry;
    }

    public CamelContext getCamelContext() {
        return contextMap.computeIfAbsent("camel-1", key -> {
            DefaultCamelContext context = new DefaultCamelContext();
            context.setName(key);

            CompositeRegistry registry = new CompositeRegistry();
            registry.addRegistry(this.registry);
            registry.addRegistry(context.getRegistry());

            context.setRegistry(registry);

            return context;
        });
    }

    public void setProperties(Properties properties) {
        PropertiesComponent pc = new PropertiesComponent();
        pc.setInitialProperties(properties);

        getRegistry().bind("properties", pc);
    }

    @Override
    protected ProducerTemplate findOrCreateCamelTemplate() {
        return getCamelContexts().size() > 0 ? getCamelContexts().get(0).createProducerTemplate() : null;
    }

    @Override
    protected Map<String, CamelContext> getCamelContextMap() {
        getCamelContext();

        return contextMap;
    }

    @Override
    protected void doStart() throws Exception {
        super.doStart();
        postProcessContext();
        if (!getCamelContexts().isEmpty()) {
            try {
                getCamelContexts().get(0).start();
            } finally {
                if (getCamelContexts().get(0).isVetoStarted()) {
                    completed();
                }
            }
        }
    }

    @Override
    protected void doStop() throws Exception {
        super.doStop();

        if (!getCamelContexts().isEmpty()) {
            getCamelContexts().get(0).stop();
        }
    }
}

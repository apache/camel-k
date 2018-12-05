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
package org.apache.camel.k.spring.boot;

import java.util.Map;
import java.util.Properties;
import java.util.Set;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.Constants;
import org.apache.camel.k.jvm.RoutesLoader;
import org.apache.camel.k.jvm.RoutesLoaders;
import org.apache.camel.k.jvm.RuntimeRegistry;
import org.apache.camel.k.jvm.RuntimeSupport;
import org.apache.camel.k.jvm.Source;
import org.apache.camel.spi.Registry;
import org.apache.camel.spring.boot.CamelContextConfiguration;
import org.apache.camel.util.ObjectHelper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.support.PropertySourcesPlaceholderConfigurer;

@SpringBootApplication
public class Application {
    private static final Logger LOGGER = LoggerFactory.getLogger(Application.class);

    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }

    // *****************************
    //
    // Beans
    //
    // *****************************

    @Bean
    public static PropertySourcesPlaceholderConfigurer propertySourcesPlaceholderConfigurer() {
        // load properties using default behaviour
        final Properties properties = RuntimeSupport.loadProperties();

        // set spring boot specific properties
        properties.put("camel.springboot.main-run-controller", "true");
        properties.put("camel.springboot.name", "camel-1");
        properties.put("camel.springboot.streamCachingEnabled", "true");
        properties.put("camel.springboot.xml-routes", "false");
        properties.put("camel.springboot.xml-rests", "false");
        properties.put("camel.springboot.jmx-enabled", "false");

        // set loaded properties as default properties
        PropertySourcesPlaceholderConfigurer configurer = new PropertySourcesPlaceholderConfigurer();
        configurer.setProperties(properties);

        return configurer;
    }

    @Bean
    public CamelContextConfiguration routesConfiguration(ConfigurableApplicationContext applicationContext) throws Exception {
        return new CamelContextConfiguration() {
            @Override
            public void beforeApplicationStart(CamelContext context) {
                final RuntimeRegistry registry = new RuntimeApplicationContextRegistry(applicationContext, context.getRegistry());
                final String routes = System.getenv(Constants.ENV_CAMEL_K_ROUTES);

                if (ObjectHelper.isEmpty(routes)) {
                    throw new IllegalStateException("No valid routes found in " + Constants.ENV_CAMEL_K_ROUTES + " environment variable");
                }

                try {
                    for (String route : routes.split(",")) {
                        final Source source = Source.create(route);
                        final RoutesLoader loader = RoutesLoaders.loaderFor(source);
                        final RouteBuilder builder = loader.load(registry, source);

                        if (builder == null) {
                            throw new IllegalStateException("Unable to load route from: " + route);
                        }

                        LOGGER.info("Routes: {}", route);

                        context.addRoutes(builder);
                    }
                } catch (Exception e) {
                    throw new IllegalStateException(e);
                }
            }

            @Override
            public void afterApplicationStart(CamelContext camelContext) {
            }
        };
    }

    // *****************************
    //
    // Registry
    //
    // *****************************

    private static class RuntimeApplicationContextRegistry implements RuntimeRegistry {
        private final ConfigurableApplicationContext applicationContext;
        private final Registry registry;

        public RuntimeApplicationContextRegistry(ConfigurableApplicationContext applicationContext, Registry registry) {
            this.applicationContext = applicationContext;
            this.registry = registry;
        }

        @Override
        public Object lookupByName(String name) {
            return registry.lookupByName(name);
        }

        @Override
        public <T> T lookupByNameAndType(String name, Class<T> type) {
            return registry.lookupByNameAndType(name, type);
        }

        @Override
        public <T> Map<String, T> findByTypeWithName(Class<T> type) {
            return registry.findByTypeWithName(type);
        }

        @Override
        public <T> Set<T> findByType(Class<T> type) {
            return registry.findByType(type);
        }
        @Override
        public void bind(String name, Object bean) {
            applicationContext.getBeanFactory().registerSingleton(name, bean);
        }
    }

}

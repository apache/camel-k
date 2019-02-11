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

import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Properties;
import java.util.Set;

import org.apache.camel.CamelContext;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.listener.ContextConfigurer;
import org.apache.camel.k.listener.RoutesConfigurer;
import org.apache.camel.k.listener.RoutesDumper;
import org.apache.camel.k.support.RuntimeSupport;
import org.apache.camel.spring.boot.CamelContextConfiguration;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.support.PropertySourcesPlaceholderConfigurer;

@Configuration
public class ApplicationAutoConfiguration {

    @Bean
    public CamelContextConfiguration routesConfiguration(ConfigurableApplicationContext applicationContext) throws Exception {
        return new CamelContextConfigurer(applicationContext, Arrays.asList(new ContextConfigurer(), new RoutesConfigurer(), new RoutesDumper()));
    }

    // *****************************
    //
    //
    //
    // *****************************

    private static class CamelContextConfigurer implements CamelContextConfiguration {
        private final ConfigurableApplicationContext applicationContext;
        private final List<Runtime.Listener> listeners;

        public CamelContextConfigurer(ConfigurableApplicationContext applicationContext, List<Runtime.Listener> listeners) {
            this.applicationContext = applicationContext;
            this.listeners = listeners;
        }

        @Override
        public void beforeApplicationStart(CamelContext context) {
            final Runtime.Registry registry = new RuntimeApplicationContextRegistry(applicationContext, context.getRegistry());
            final Runtime runtime = new Runtime() {
                @Override
                public CamelContext getContext() {
                    return context;
                }
                @Override
                public Registry getRegistry() {
                    return registry;
                }
            };

            listeners.forEach(l -> l.accept(Runtime.Phase.Starting, runtime));
            listeners.forEach(l -> l.accept(Runtime.Phase.ConfigureContext, runtime));
            listeners.forEach(l -> l.accept(Runtime.Phase.ConfigureRoutes, runtime));
        }

        @Override
        public void afterApplicationStart(CamelContext context) {
            final Runtime.Registry registry = new RuntimeApplicationContextRegistry(applicationContext, context.getRegistry());
            final Runtime runtime = new Runtime() {
                @Override
                public CamelContext getContext() {
                    return context;
                }
                @Override
                public Registry getRegistry() {
                    return registry;
                }
            };

            listeners.forEach(l -> l.accept(Runtime.Phase.Started, runtime));

        }
    }

    private static class RuntimeApplicationContextRegistry implements Runtime.Registry {
        private final ConfigurableApplicationContext applicationContext;
        private final org.apache.camel.spi.Registry registry;

        public RuntimeApplicationContextRegistry(ConfigurableApplicationContext applicationContext, org.apache.camel.spi.Registry registry) {
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

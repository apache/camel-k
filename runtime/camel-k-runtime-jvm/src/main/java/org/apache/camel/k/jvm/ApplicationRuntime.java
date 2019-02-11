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

import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

import org.apache.camel.CamelContext;
import org.apache.camel.ProducerTemplate;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.camel.k.InMemoryRegistry;
import org.apache.camel.k.Runtime;
import org.apache.camel.main.MainSupport;
import org.apache.camel.util.ObjectHelper;
import org.apache.camel.util.function.ThrowingConsumer;

public final class ApplicationRuntime implements Runtime {
    private final Main main;
    private final ConcurrentMap<String, CamelContext> contextMap;
    private final Runtime.Registry registry;
    private final Set<Runtime.Listener> listeners;

    public ApplicationRuntime() {
        this.contextMap = new ConcurrentHashMap<>();
        this.registry = new InMemoryRegistry();
        this.listeners = new LinkedHashSet<>();

        this.main = new Main();
        this.main.addMainListener(new MainListenerAdapter());
    }

    @Override
    public CamelContext getContext() {
        return contextMap.computeIfAbsent("camel-k", key -> {
            DefaultCamelContext context = new DefaultCamelContext();
            context.setName(key);
            context.setRegistry(this.registry);

            return context;
        });
    }

    @Override
    public Runtime.Registry getRegistry() {
        return registry;
    }

    public void run() throws Exception {
        this.main.run();
    }

    public void stop()throws Exception {
        this.main.stop();
    }

    public void addListener(Runtime.Listener listener) {
        this.listeners.add(listener);
    }

    public void addListener(Phase phase, ThrowingConsumer<Runtime, Exception> consumer) {
        addListener((p, runtime) -> {
            if (p == phase) {
                try {
                    consumer.accept(runtime);
                } catch (Exception e) {
                    throw ObjectHelper.wrapRuntimeCamelException(e);
                }
            }
        });
    }

    private class Main extends org.apache.camel.main.MainSupport {
        @Override
        protected ProducerTemplate findOrCreateCamelTemplate() {
            return getContext().createProducerTemplate();
        }

        @Override
        protected Map<String, CamelContext> getCamelContextMap() {
            getContext();

            return contextMap;
        }

        @Override
        protected void doStart() throws Exception {
            super.doStart();
            postProcessContext();

            try {
                getContext().start();
            } finally {
                if (getContext().isVetoStarted()) {
                    completed();
                }
            }
        }

        @Override
        protected void doStop() throws Exception {
            super.doStop();

            if (!getCamelContexts().isEmpty()) {
                getContext().stop();
            }
        }
    }

    private class MainListenerAdapter implements org.apache.camel.main.MainListener {

        @Override
        public void beforeStart(MainSupport main) {
            listeners.forEach(l -> l.accept(Phase.Starting, ApplicationRuntime.this));
        }

        @Override
        public void configure(CamelContext context) {
            listeners.forEach(l -> l.accept(Phase.ConfigureContext, ApplicationRuntime.this));
            listeners.forEach(l -> l.accept(Phase.ConfigureRoutes, ApplicationRuntime.this));
        }

        @Override
        public void afterStart(MainSupport main) {
            listeners.forEach(l -> l.accept(Phase.Started, ApplicationRuntime.this));
        }

        @Override
        public void beforeStop(MainSupport main) {

        }

        @Override
        public void afterStop(MainSupport main) {

        }
    }
}

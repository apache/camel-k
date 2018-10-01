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
package org.apache.camel.k.groovy;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.lang.reflect.Array;
import java.util.Collections;
import java.util.List;

import groovy.lang.Binding;
import groovy.lang.Closure;
import groovy.lang.GroovyObjectSupport;
import groovy.lang.GroovyShell;
import groovy.util.DelegatingScript;
import org.apache.camel.CamelContext;
import org.apache.camel.Component;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.Language;
import org.apache.camel.k.jvm.RoutesLoader;
import org.apache.camel.k.jvm.RuntimeRegistry;
import org.apache.camel.k.jvm.dsl.Components;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.rest.RestConfigurationDefinition;
import org.apache.camel.model.rest.RestDefinition;
import org.apache.camel.util.IntrospectionSupport;
import org.apache.camel.util.ResourceHelper;
import org.codehaus.groovy.control.CompilerConfiguration;

public class GroovyRoutesLoader implements RoutesLoader {
    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.Groovy);
    }

    @Override
    public RouteBuilder load(RuntimeRegistry registry, String resource) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                CompilerConfiguration cc = new CompilerConfiguration();
                cc.setScriptBaseClass(DelegatingScript.class.getName());

                ClassLoader cl = Thread.currentThread().getContextClassLoader();
                GroovyShell sh = new GroovyShell(cl, new Binding(), cc);

                try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(getContext(), resource)) {
                    Reader reader = new InputStreamReader(is);
                    DelegatingScript script = (DelegatingScript) sh.parse(reader);

                    // set the delegate target
                    script.setDelegate(new Scripting(registry, this));
                    script.run();
                }
            }
        };
    }


    public static class Scripting  {
        private final RuntimeRegistry registry;

        public final CamelContext context;
        public final Components components;
        public final RouteBuilder builder;

        public Scripting(RuntimeRegistry registry, RouteBuilder builder) {
            this.registry = registry;
            this.context = builder.getContext();
            this.components = new Components(this.context);
            this.builder = builder;
        }

        public Component component(String name, Closure<Component> callable) {
            final Component component = context.getComponent(name, true);

            callable.setResolveStrategy(Closure.DELEGATE_ONLY);
            callable.setDelegate(new GroovyObjectSupport() {
                public Object invokeMethod(String name, Object arg) {
                    if (arg == null) {
                        return super.invokeMethod(name, arg);
                    }
                    if (!arg.getClass().isArray()) {
                        return super.invokeMethod(name, arg);
                    }

                    try {
                        IntrospectionSupport.setProperty(component, name, Array.get(arg, 0), true);
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }

                    return component;
                }
            });

            return callable.call();
        }

        public RouteDefinition from(String endpoint) {
            return builder.from(endpoint);
        }

        public RestDefinition rest() {
            return builder.rest();
        }

        public RestDefinition rest(Closure<RestDefinition> callable) {
            callable.setResolveStrategy(Closure.DELEGATE_ONLY);
            callable.setDelegate(builder.rest());
            return callable.call();
        }

        public RestConfigurationDefinition restConfiguration() {
            return builder.restConfiguration();
        }

        public void restConfiguration(Closure<?> callable) {
            callable.setResolveStrategy(Closure.DELEGATE_ONLY);
            callable.setDelegate(builder.restConfiguration());
            callable.call();
        }

        public RestConfigurationDefinition restConfiguration(String component, Closure<RestConfigurationDefinition> callable) {
            callable.setResolveStrategy(Closure.DELEGATE_ONLY);
            callable.setDelegate(builder.restConfiguration(component));
            return callable.call();
        }

        public RuntimeRegistry registry(Closure<RuntimeRegistry> callable) {
            callable.setResolveStrategy(Closure.DELEGATE_ONLY);
            callable.setDelegate(registry);

            return callable.call();
        }
    }
}

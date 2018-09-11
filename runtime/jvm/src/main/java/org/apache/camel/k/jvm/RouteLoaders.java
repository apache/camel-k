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

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.function.Function;
import javax.script.Bindings;
import javax.script.ScriptEngine;
import javax.script.ScriptEngineManager;
import javax.script.SimpleBindings;

import groovy.lang.Binding;
import groovy.lang.GroovyShell;
import groovy.util.DelegatingScript;
import org.apache.camel.CamelContext;
import org.apache.camel.Component;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.RouteDefinition;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.codehaus.groovy.control.CompilerConfiguration;
import org.joor.Reflect;

public enum RouteLoaders implements RoutesLoader {
    JavaClass {
        @Override
        public boolean test(String resource) {
            return !isScripting(resource) && hasSupportedScheme(resource);
        }

        @Override
        public RouteBuilder load(String resource) throws Exception {
            String path = resource.substring(Application.SCHEME_CLASSPATH.length());
            Class<?> type = Class.forName(path);

            if (!RouteBuilder.class.isAssignableFrom(type)) {
                throw new IllegalStateException("The class provided (" + path + ") is not a org.apache.camel.builder.RouteBuilder");
            }

            return (RouteBuilder)type.newInstance();
        }
    },
    JavaSource {
        @Override
        public boolean test(String resource) {
            return isScripting(resource, "java") && hasSupportedScheme(resource);
        }

        @Override
        public RouteBuilder load(String resource) throws Exception {
            try (InputStream is = is(resource)) {
                String name = StringUtils.substringAfter(resource, ":");
                name = StringUtils.removeEnd(name, ".java");
                name = StringUtils.removeStart(name, "/");

                return Reflect.compile(name.replace("/", "."), IOUtils.toString(is)).create().get();
            }
        }
    },
    JavaScript {
        @Override
        public boolean test(String resource) {
            return isScripting(resource, "js") && hasSupportedScheme(resource);
        }

        @Override
        public RouteBuilder load(String resource) throws Exception {
            return new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    final CamelContext context = getContext();
                    final ScriptEngineManager manager = new ScriptEngineManager();
                    final ScriptEngine engine = manager.getEngineByName("nashorn");
                    final Bindings bindings = new SimpleBindings();

                    // Exposed to the underlying script, but maybe better to have
                    // a nice dsl
                    bindings.put("context", context);
                    bindings.put("components", context);
                    bindings.put("from", (Function<String, RouteDefinition>) uri -> from(uri));

                    try (InputStream is = is(resource)) {
                        engine.eval(new InputStreamReader(is), bindings);
                    }
                }
            };
        }
    },
    Groovy {
        @Override
        public boolean test(String resource) {
            return isScripting(resource, "groovy") && hasSupportedScheme(resource);
        }

        @Override
        public RouteBuilder load(String resource) throws Exception {
            return new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    CompilerConfiguration cc = new CompilerConfiguration();
                    cc.setScriptBaseClass(DelegatingScript.class.getName());

                    ClassLoader cl = Thread.currentThread().getContextClassLoader();
                    GroovyShell sh = new GroovyShell(cl, new Binding(), cc);

                    try (InputStream is = is(resource)) {
                        Reader reader = new InputStreamReader(is);
                        DelegatingScript script = (DelegatingScript) sh.parse(reader);

                        // set the delegate target
                        script.setDelegate(new ScriptingDsl(this));
                        script.run();
                    }
                }
            };
        }
    };

    // ********************************
    //
    // Helpers
    //
    // TODO: move to a dedicate class
    // ********************************

    public static boolean isScripting(String resource, String type) {
        return type.startsWith(".") ? resource.endsWith(type) : resource.endsWith("." + type);
    }

    public static boolean isScripting(String resource) {
        return resource.endsWith(".java") || resource.endsWith(".js") || resource.endsWith(".groovy");
    }

    public static boolean hasSupportedScheme(String resource) {
        return resource.startsWith(Application.SCHEME_CLASSPATH) || resource.startsWith(Application.SCHEME_FILE);
    }

    public static RoutesLoader loaderFor(String resource) {
        for (RoutesLoader loader: RouteLoaders.values()) {
            if (loader.test(resource)) {
                return loader;
            }
        }

        throw new IllegalArgumentException("Unable to find loader for: " + resource);
    }

    private static InputStream is(String resource) throws IOException {
        if (resource.startsWith(Application.SCHEME_CLASSPATH)) {
            return Application.class.getResourceAsStream(
                resource.substring(Application.SCHEME_CLASSPATH.length())
            );
        } else {
            return Files.newInputStream(
                Paths.get(resource.substring(Application.SCHEME_FILE.length()))
            );
        }
    }

    public static class Components {
        private CamelContext context;

        public Components(CamelContext context) {
            this.context = context;
        }

        public Component get(String scheme) {
            return context.getComponent(scheme, true);
        }

        public Component put(String scheme, Component instance) {
            context.addComponent(scheme, instance);

            return instance;
        }

        public Component make(String scheme, String type) {
            final Class<?> clazz = context.getClassResolver().resolveClass(type);
            final Component instance = (Component)context.getInjector().newInstance(clazz);

            context.addComponent(scheme, instance);

            return instance;
        }
    }

    private static class ScriptingDsl {
        private final RouteBuilder builder;

        public final CamelContext context;
        public final Components components;

        public ScriptingDsl(RouteBuilder builder) {
            this.builder = builder;
            this.context = builder.getContext();
            this.components = new Components(builder.getContext());
        }

        public RouteDefinition from(String endpoint) {
            return builder.from(endpoint);
        }
    }
}

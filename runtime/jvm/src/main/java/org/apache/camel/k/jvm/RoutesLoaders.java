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

import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.List;
import java.util.ServiceLoader;
import java.util.function.Function;
import java.util.function.Supplier;
import javax.script.Bindings;
import javax.script.ScriptEngine;
import javax.script.ScriptEngineManager;
import javax.script.SimpleBindings;
import javax.xml.bind.UnmarshalException;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.dsl.Components;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.rest.RestConfigurationDefinition;
import org.apache.camel.model.rest.RestDefinition;
import org.apache.camel.util.ObjectHelper;
import org.apache.camel.util.ResourceHelper;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.joor.Reflect;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import static org.apache.camel.k.jvm.Constants.SCHEME_CLASSPATH;
import static org.apache.camel.k.jvm.Constants.SCHEME_FILE;

public final class RoutesLoaders {
    private static final Logger LOGGER = LoggerFactory.getLogger(RoutesLoaders.class);

    private RoutesLoaders() {
    }

    public static class JavaClass implements RoutesLoader {
        @Override
        public List<Language> getSupportedLanguages() {
            return Collections.singletonList(Language.JavaClass);
        }

        @Override
        public RouteBuilder load(RuntimeRegistry registry, String resource) throws Exception {
            String path = resource;
            path = StringUtils.removeStart(path, SCHEME_CLASSPATH);
            path = StringUtils.removeEnd(path, ".class");

            Class<?> type = Class.forName(path);

            if (!RouteBuilder.class.isAssignableFrom(type)) {
                throw new IllegalStateException("The class provided (" + path + ") is not a org.apache.camel.builder.RouteBuilder");
            }

            return (RouteBuilder)type.newInstance();
        }
    }

    public static class JavaSource implements RoutesLoader {
        @Override
        public List<Language> getSupportedLanguages() {
            return Collections.singletonList(Language.JavaSource);
        }

        @Override
        public RouteBuilder load(RuntimeRegistry registry, String resource) throws Exception {
            return new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(getContext(), resource)) {
                        String name = StringUtils.substringAfter(resource, ":");
                        name = StringUtils.removeEnd(name, ".java");

                        if (name.contains("/")) {
                            name = StringUtils.substringAfterLast(name, "/");
                        }

                        // Wrap routes builder
                        includeRoutes(
                            Reflect.compile(name, IOUtils.toString(is, StandardCharsets.UTF_8)).create().get()
                        );
                    }
                }
            };
        }
    }

    public static class JavaScript implements RoutesLoader {
        @Override
        public List<Language> getSupportedLanguages() {
            return Collections.singletonList(Language.JavaScript);
        }

        @Override
        public RouteBuilder load(RuntimeRegistry registry, String resource) throws Exception {
            return new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    final CamelContext context = getContext();
                    final ScriptEngineManager manager = new ScriptEngineManager();
                    final ScriptEngine engine = manager.getEngineByName("nashorn");
                    final Bindings bindings = new SimpleBindings();

                    // Exposed to the underlying script, but maybe better to have
                    // a nice dsl
                    bindings.put("builder", this);
                    bindings.put("context", context);
                    bindings.put("components", new Components(context));
                    bindings.put("registry", registry);
                    bindings.put("from", (Function<String, RouteDefinition>) uri -> from(uri));
                    bindings.put("rest", (Supplier<RestDefinition>) () -> rest());
                    bindings.put("restConfiguration", (Supplier<RestConfigurationDefinition>) () -> restConfiguration());

                    try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(context, resource)) {
                        engine.eval(new InputStreamReader(is), bindings);
                    }
                }
            };
        }
    }

    public static class Xml implements RoutesLoader {
        @Override
        public List<Language> getSupportedLanguages() {
            return Collections.singletonList(Language.Xml);
        }

        @Override
        public RouteBuilder load(RuntimeRegistry registry, String resource) throws Exception {
            return new RouteBuilder() {
                @Override
                public void configure() throws Exception {
                    try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(getContext(), resource)) {
                        try {
                            setRouteCollection(
                                getContext().loadRoutesDefinition(is)
                            );
                        } catch (UnmarshalException e) {
                            LOGGER.debug("Unable to load RoutesDefinition: {}", e.getMessage());
                        }

                        try {
                            setRestCollection(
                                getContext().loadRestsDefinition(is)
                            );
                        } catch (UnmarshalException e) {
                            LOGGER.debug("Unable to load RestsDefinition: {}", e.getMessage());
                        }
                    }
                }
            };
        }
    }


    public static RoutesLoader loaderFor(String resource, String languageName) {
        if (!resource.startsWith(SCHEME_CLASSPATH) && !resource.startsWith(SCHEME_FILE)) {
            throw new IllegalArgumentException("No valid resource format, expected scheme:path, found " + resource);
        }

        Language language = ObjectHelper.isNotEmpty(languageName)
            ? Language.fromLanguageName(languageName)
            : Language.fromResource(resource);

        for (RoutesLoader loader: ServiceLoader.load(RoutesLoader.class)) {
            if (loader.getSupportedLanguages().contains(language)) {
                return loader;
            }
        }

        throw new IllegalArgumentException("Unable to find loader for: resource=" + resource + " language=" + languageName);
    }
}

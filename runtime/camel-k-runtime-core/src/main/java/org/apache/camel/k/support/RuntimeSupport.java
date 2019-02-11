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
package org.apache.camel.k.support;

import java.io.IOException;
import java.io.Reader;
import java.nio.file.FileVisitResult;
import java.nio.file.FileVisitor;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.Objects;
import java.util.Properties;
import java.util.concurrent.atomic.AtomicInteger;

import org.apache.camel.CamelContext;
import org.apache.camel.NoFactoryAvailableException;
import org.apache.camel.component.properties.PropertiesComponent;
import org.apache.camel.k.Constants;
import org.apache.camel.k.ContextCustomizer;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.Source;
import org.apache.camel.spi.FactoryFinder;
import org.apache.camel.spi.RestConfiguration;
import org.apache.camel.util.IntrospectionSupport;
import org.apache.camel.util.ObjectHelper;
import org.apache.commons.io.FilenameUtils;


public final class RuntimeSupport {

    private RuntimeSupport() {
    }

    public static void configureContext(CamelContext context, Runtime.Registry registry) {
        try {
            FactoryFinder finder = context.getFactoryFinder(Constants.CONTEXT_CUSTOMIZER_RESOURCE_PATH);
            String customizerIDs = System.getenv().getOrDefault(Constants.ENV_CAMEL_K_CUSTOMIZERS, "");

            if (ObjectHelper.isEmpty(customizerIDs)) {
                PropertiesComponent component = context.getComponent("properties", PropertiesComponent.class);
                Properties properties = component.getInitialProperties();

                if (properties != null) {
                    customizerIDs = properties.getProperty(Constants.PROPERTY_CAMEL_K_CUSTOMIZER, "");
                }
            }

            if  (ObjectHelper.isNotEmpty(customizerIDs)) {
                for (String customizerId : customizerIDs.split(",", -1)) {
                    configureContext(context, customizerId, (ContextCustomizer) finder.newInstance(customizerId), registry);
                }
            }
        } catch (NoFactoryAvailableException e) {
            // ignored
        }

        //this is to initialize all customizers that might be already present in the context injected by other means.
        context.getRegistry().findByTypeWithName(ContextCustomizer.class).forEach(
            (customizerId, customizer) -> configureContext(context, customizerId, customizer, registry)
        );
    }

    public static void configureContext(CamelContext context, String customizerId, ContextCustomizer customizer, Runtime.Registry registry) {
        bindProperties(context, customizer, "customizer." + customizerId + ".");
        customizer.apply(context, registry);
    }

    public static void configureRest(CamelContext context) {
        RestConfiguration configuration = new RestConfiguration();

        if (RuntimeSupport.bindProperties(context, configuration, "camel.rest.") > 0) {
            //
            // Set the rest configuration if only if at least one
            // rest parameter has been set.
            //
            context.setRestConfiguration(configuration);
        }
    }

    public static int bindProperties(CamelContext context, Object target, String prefix) {
        final PropertiesComponent component = context.getComponent("properties", PropertiesComponent.class);
        final Properties properties = component.getInitialProperties();

        if (properties == null) {
            return 0;
        }

        return bindProperties(properties, target, prefix);
    }

    public static int bindProperties(Properties properties, Object target, String prefix) {
        final AtomicInteger count = new AtomicInteger();

        properties.entrySet().stream()
            .filter(entry -> entry.getKey() instanceof String)
            .filter(entry -> entry.getValue() != null)
            .filter(entry -> ((String)entry.getKey()).startsWith(prefix))
            .forEach(entry -> {
                    final String key = ((String)entry.getKey()).substring(prefix.length());
                    final Object val = entry.getValue();

                    try {
                        if (IntrospectionSupport.setProperty(target, key, val, false)) {
                            count.incrementAndGet();
                        }
                    } catch (Exception ex) {
                        throw new RuntimeException(ex);
                    }
                }
            );

        return count.get();
    }

    public static RoutesLoader loaderFor(CamelContext context, Source source) {
        return  context.getRegistry().findByType(RoutesLoader.class).stream()
            .filter(rl -> rl.getSupportedLanguages().contains(source.getLanguage()))
            .findFirst()
            .orElseGet(() -> lookupLoaderFromResource(context, source));
    }

    public static RoutesLoader lookupLoaderFromResource(CamelContext context, Source source) {
        final FactoryFinder finder;
        final RoutesLoader loader;

        try {
            finder = context.getFactoryFinder(Constants.ROUTES_LOADER_RESOURCE_PATH);
            loader = (RoutesLoader)finder.newInstance(source.getLanguage());
        } catch (NoFactoryAvailableException e) {
            throw new IllegalArgumentException("Unable to find loader for: " + source, e);
        }

        return loader;
    }

    public static Properties loadProperties() {
        final String conf = System.getenv(Constants.ENV_CAMEL_K_CONF);
        final String confd = System.getenv(Constants.ENV_CAMEL_K_CONF_D);

        return loadProperties(conf, confd);
    }

    public static Properties loadProperties(String conf, String confd) {
        final Properties properties = new Properties();

        // Main location
        if (ObjectHelper.isNotEmpty(conf)) {
            if (conf.startsWith(Constants.SCHEME_ENV)) {
                try (Reader reader = URIResolver.resolveEnv(conf)) {
                    properties.load(reader);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            } else {
                try (Reader reader = Files.newBufferedReader(Paths.get(conf))) {
                    properties.load(reader);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }
        }

        // Additional locations
        if (ObjectHelper.isNotEmpty(confd)) {
            Path root = Paths.get(confd);
            FileVisitor<Path> visitor = new SimpleFileVisitor<Path>() {
                @Override
                public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
                    Objects.requireNonNull(file);
                    Objects.requireNonNull(attrs);

                    String path = file.toFile().getAbsolutePath();
                    String ext = FilenameUtils.getExtension(path);

                    if (Objects.equals("properties", ext)) {
                        try (Reader reader = Files.newBufferedReader(Paths.get(path))) {
                            properties.load(reader);
                        }
                    }

                    return FileVisitResult.CONTINUE;
                }
            };

            if (Files.exists(root)) {
                try {
                    Files.walkFileTree(root, visitor);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }
        }

        return properties;
    }
}

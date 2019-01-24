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

import java.util.Properties;
import java.util.concurrent.atomic.AtomicInteger;

import org.apache.camel.CamelContext;
import org.apache.camel.NoFactoryAvailableException;
import org.apache.camel.component.properties.PropertiesComponent;
import org.apache.camel.k.Constants;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeTrait;
import org.apache.camel.k.Source;
import org.apache.camel.spi.FactoryFinder;
import org.apache.camel.spi.RestConfiguration;
import org.apache.camel.util.IntrospectionSupport;
import org.apache.camel.util.ObjectHelper;


public final class RuntimeSupport {
    private RuntimeSupport() {
    }

    public static void configureContext(CamelContext context) {
        try {
            FactoryFinder finder = context.getFactoryFinder(Constants.RUNTIME_TRAIT_RESOURCE_PATH);
            String traitIDs = System.getenv().getOrDefault(Constants.ENV_CAMEL_K_TRAITS, "");

            if (ObjectHelper.isEmpty(traitIDs)) {
                PropertiesComponent component = context.getComponent("properties", PropertiesComponent.class);
                Properties properties = component.getInitialProperties();

                traitIDs = properties.getProperty("camel.k.traits", "");
            }

            for (String traitId: traitIDs.split(",", -1)) {
                configureContext(context, traitId, (RuntimeTrait)finder.newInstance(traitId));
            }
        } catch (NoFactoryAvailableException e) {
            // ignored
        }

        //this is to initialize all traits that might be already present in the context injected by other means.
        context.getRegistry().findByTypeWithName(RuntimeTrait.class).forEach(
            (traitId, trait) -> configureContext(context, traitId, trait)
        );
    }

    public static void configureContext(CamelContext context, String traitId, RuntimeTrait trait) {
        bindProperties(context, trait, "trait." + traitId + ".");
        trait.apply(context);
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
            throw new IllegalStateException("PropertiesComponent has no properties");
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
            loader = (RoutesLoader)finder.newInstance(source.getLanguage().getId());
        } catch (NoFactoryAvailableException e) {
            throw new IllegalArgumentException("Unable to find loader for: " + source, e);
        }

        return loader;
    }
}

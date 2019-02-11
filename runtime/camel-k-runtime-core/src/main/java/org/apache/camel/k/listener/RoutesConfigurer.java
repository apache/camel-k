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
package org.apache.camel.k.listener;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.Constants;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.Source;
import org.apache.camel.k.support.RuntimeSupport;
import org.apache.camel.util.ObjectHelper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class RoutesConfigurer extends AbstractPhaseListener {
    private static final Logger LOGGER = LoggerFactory.getLogger(RoutesConfigurer.class);

    public RoutesConfigurer() {
        super(Runtime.Phase.ConfigureRoutes);
    }

    @Override
    protected void accept(Runtime runtime) {
        final String routes = System.getenv(Constants.ENV_CAMEL_K_ROUTES);

        if (ObjectHelper.isEmpty(routes)) {
            LOGGER.warn("No valid routes found in {} environment variable", Constants.ENV_CAMEL_K_ROUTES);
        }

        load(runtime, routes.split(",", -1));
    }

    protected void load(Runtime runtime, String[] routes) {
        for (String route: routes) {
            final Source source;
            final RoutesLoader loader;
            final RouteBuilder builder;

            try {
                source = Source.create(route);
                loader = RuntimeSupport.loaderFor(runtime.getContext(), source);
                builder = loader.load(runtime.getRegistry(), source);
            } catch (Exception e) {
                throw ObjectHelper.wrapRuntimeCamelException(e);
            }

            if (builder == null) {
                throw new IllegalStateException("Unable to load route from: " + route);
            }

            LOGGER.info("Loading routes from: {}", route);

            try {
                runtime.getContext().addRoutes(builder);
            } catch (Exception e) {
                throw ObjectHelper.wrapRuntimeCamelException(e);
            }
        }
    }

    public static RoutesConfigurer forRoutes(String... routes) {
        return new RoutesConfigurer() {
            @Override
            protected void accept(Runtime runtime) {
                load(runtime, routes);
            }
        };
    }
}

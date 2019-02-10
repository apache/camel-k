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

import org.apache.camel.CamelContext;
import org.apache.camel.Component;
import org.apache.camel.RuntimeCamelException;
import org.apache.camel.k.Constants;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.support.RuntimeSupport;
import org.apache.camel.main.MainListenerSupport;
import org.apache.camel.main.MainSupport;
import org.apache.camel.model.ModelHelper;
import org.apache.camel.model.RoutesDefinition;
import org.apache.camel.model.rest.RestsDefinition;
import org.apache.camel.support.LifecycleStrategySupport;
import org.apache.camel.util.ObjectHelper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.xml.bind.JAXBException;

public class Application {
    static {
        //
        // Configure the logging subsystem log4j2 using a subset of spring boot
        // conventions:
        //
        //    logging.level.${nane} = OFF|FATAL|ERROR|WARN|INFO|DEBUG|TRACE|ALL
        //
        // We now support setting the logging level only
        //
        ApplicationSupport.configureLogging();

        //
        // Install a custom protocol handler to support discovering resources
        // from the platform i.e. in knative, resources are provided through
        // env var as it is not possible to mount config maps / secrets.
        //
        ApplicationSupport.configureStreamHandler();
    }

    // *******************************
    //
    // Main
    //
    // *******************************

    public static void main(String[] args) throws Exception {
        final String routes = System.getenv(Constants.ENV_CAMEL_K_ROUTES);

        if (ObjectHelper.isEmpty(routes)) {
            throw new IllegalStateException("No valid routes found in " + Constants.ENV_CAMEL_K_ROUTES + " environment variable");
        }

        Runtime runtime = new Runtime();
        runtime.setProperties(ApplicationSupport.loadProperties());
        runtime.addMainListener(new CamelkJvmRuntimeConfigurer(runtime, routes.split(",", -1)));
        runtime.addMainListener(new RoutesDumper());

        runtime.run();
    }

    // *******************************
    //
    // Listeners
    //
    // *******************************

    static class CamelkJvmRuntimeConfigurer extends MainListenerSupport {

        private RuntimeRegistry registry;
        private Runtime runtime;
        private String[] routes;

        public CamelkJvmRuntimeConfigurer(Runtime runtime, String[] routes){
            this.runtime = runtime;
            this.registry = runtime.getRegistry();
            this.routes = routes;
        }

        @Override
        public void configure(CamelContext context) {
            //
            // Configure the camel context using properties in the form:
            //
            //     camel.context.${name} = ${value}
            //
            RuntimeSupport.bindProperties(context, context, "camel.context.");

            //
            // Configure the camel rest definition using properties in the form:
            //
            //     camel.rest.${name} = ${value}
            //
            RuntimeSupport.configureRest(context);

            //
            // Programmatically configure the camel context.
            //
            // This is useful to configure services such as the ClusterService,
            // RouteController, etc
            //
            RuntimeSupport.configureContext(context, registry);

            //
            // Configure components upon creation
            //
            context.addLifecycleStrategy(new LifecycleStrategySupport() {
                @SuppressWarnings("unchecked")
                @Override
                public void onComponentAdd(String name, Component component) {
                    // The prefix that identifies component properties is the
                    // same one used by camel-spring-boot to configure components
                    // using starters:
                    //
                    //     camel.component.${scheme}.${name} = ${value}
                    //
                    RuntimeSupport.bindProperties(context, component, "camel.component." + name + ".");
                }
            });

            //
            // Load routes
            //
            try {
                runtime.load(routes);
            } catch (Exception e) {
                throw new RuntimeCamelException("CamelkJvmRuntimeConfigurer has failed to load routes: "+routes);
            }
        }
    }

    static class RoutesDumper extends MainListenerSupport {
        static final Logger LOGGER = LoggerFactory.getLogger(RoutesDumper.class);

        @Override
        public void afterStart(MainSupport main) {
            CamelContext context = main.getCamelContexts().get(0);

            RoutesDefinition routes = new RoutesDefinition();
            routes.setRoutes(context.getRouteDefinitions());

            RestsDefinition rests = new RestsDefinition();
            rests.setRests(context.getRestDefinitions());

            try {
                if (LOGGER.isDebugEnabled() && !routes.getRoutes().isEmpty()) {
                    LOGGER.debug("Routes: \n{}", ModelHelper.dumpModelAsXml(context, routes));
                }
                if (LOGGER.isDebugEnabled() && !rests.getRests().isEmpty()) {
                    LOGGER.debug("Rests: \n{}", ModelHelper.dumpModelAsXml(context, rests));
                }
            } catch (JAXBException e) {
                throw new IllegalArgumentException(e);
            }
        }
    }
}

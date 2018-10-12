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
import org.apache.camel.main.MainListenerSupport;
import org.apache.camel.support.LifecycleStrategySupport;
import org.apache.camel.util.ObjectHelper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Application {
    private static final Logger LOGGER = LoggerFactory.getLogger(Application.class);

    static {
        //
        // Load properties as system properties so they are accessible through
        // camel's properties component as well as for runtime configuration.
        //
        RuntimeSupport.configureSystemProperties();

        //
        // Configure the logging subsystem log4j2 using a subset of spring boot
        // conventions:
        //
        //    logging.level.${nane} = OFF|FATAL|ERROR|WARN|INFO|DEBUG|TRACE|ALL
        //
        // We now support setting the logging level only
        //
        RuntimeSupport.configureLogging();
    }

    // *******************************
    //
    // Main
    //
    // *******************************

    public static void main(String[] args) throws Exception {
        final String resource = System.getenv(Constants.ENV_CAMEL_K_ROUTES_URI);
        final String language = System.getenv(Constants.ENV_CAMEL_K_ROUTES_LANGUAGE);

        if (ObjectHelper.isEmpty(resource)) {
            throw new IllegalStateException("No valid resource found in " + Constants.ENV_CAMEL_K_ROUTES_URI + " environment variable");
        }

        Runtime runtime = new Runtime();
        runtime.load(resource, language);
        runtime.addMainListener(new ComponentPropertiesBinder());
        runtime.run();
    }

    // *******************************
    //
    // Listeners
    //
    // *******************************

    static class ComponentPropertiesBinder extends MainListenerSupport {
        @Override
        public void configure(CamelContext context) {
            // Configure the camel context using properties in the form:
            //
            //     camel.context.${name} = ${value}
            //
            RuntimeSupport.bindProperties(context, "camel.context.");

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
                    RuntimeSupport.bindProperties(component, "camel.component." + name + ".");
                }
            });
        }
    }
}

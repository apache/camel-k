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

import javax.xml.bind.JAXBException;

import org.apache.camel.CamelContext;
import org.apache.camel.k.Runtime;
import org.apache.camel.model.ModelHelper;
import org.apache.camel.model.RoutesDefinition;
import org.apache.camel.model.rest.RestsDefinition;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class RoutesDumper extends AbstractPhaseListener {
    private static final Logger LOGGER = LoggerFactory.getLogger(RoutesDumper.class);

    public RoutesDumper() {
        super(Runtime.Phase.Started);
    }

    @Override
    protected void accept(Runtime runtime) {
        CamelContext context = runtime.getContext();

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

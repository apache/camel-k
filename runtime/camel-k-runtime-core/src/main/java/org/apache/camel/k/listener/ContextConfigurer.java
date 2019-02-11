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

import org.apache.camel.k.Runtime;
import org.apache.camel.k.support.RuntimeSupport;

public class ContextConfigurer extends AbstractPhaseListener {
    public ContextConfigurer() {
        super(Runtime.Phase.ConfigureContext);
    }

    @Override
    protected void accept(Runtime runtime) {
        //
        // Configure the camel context using properties in the form:
        //
        //     camel.context.${name} = ${value}
        //
        RuntimeSupport.bindProperties(runtime.getContext(), runtime.getContext(), "camel.context.");

        //
        // Configure the camel rest definition using properties in the form:
        //
        //     camel.rest.${name} = ${value}
        //
        RuntimeSupport.configureRest(runtime.getContext());

        //
        // Programmatically configure the camel context.
        //
        // This is useful to configure services such as the ClusterService,
        // RouteController, etc
        //
        RuntimeSupport.configureContext(runtime.getContext(), runtime.getRegistry());
    }
}

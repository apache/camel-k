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

import org.apache.camel.Component;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.support.RuntimeSupport;
import org.apache.camel.support.LifecycleStrategySupport;

public class ContextLifecycleConfigurer extends AbstractPhaseListener {
    public ContextLifecycleConfigurer() {
        super(Runtime.Phase.ConfigureContext);
    }

    @Override
    protected void accept(Runtime runtime) {
        //
        // Configure components upon creation
        //
        runtime.getContext().addLifecycleStrategy(new LifecycleStrategySupport() {
            @SuppressWarnings("unchecked")
            @Override
            public void onComponentAdd(String name, Component component) {
                // The prefix that identifies component properties is the
                // same one used by camel-spring-boot to configure components
                // using starters:
                //
                //     camel.component.${scheme}.${name} = ${value}
                //
                RuntimeSupport.bindProperties(runtime.getContext(), component, "camel.component." + name + ".");
            }
        });
    }
}

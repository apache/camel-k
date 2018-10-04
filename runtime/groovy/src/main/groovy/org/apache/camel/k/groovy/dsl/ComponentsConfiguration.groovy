/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License") you may not use this file except in compliance with
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
package org.apache.camel.k.groovy.dsl

import org.apache.camel.CamelContext
import org.apache.camel.Component

import java.lang.reflect.Array

class ComponentsConfiguration {
    private final CamelContext context

    ComponentsConfiguration(CamelContext context) {
        this.context = context
    }

    def component(String name, Closure<?> callable) {
        def component = context.getComponent(name, true, false)

        callable.resolveStrategy = Closure.DELEGATE_FIRST
        callable.delegate = new ComponentConfiguration(component)
        callable.call()
    }

    def component(String name, Class<? extends Component> type, Closure <?> callable) {
        def component = context.getComponent(name, true, false)

        // if the component is not found, let's create a new one. This is
        // equivalent to create a new named component, useful to create
        // multiple instances of the same component but with different setup
        if (component == null) {
            component = context.injector.newInstance(type)

            // let's the camel context be aware of the new component
            context.addComponent(name, component)
        }

        if (type.isAssignableFrom(component.class)) {
            callable.resolveStrategy = Closure.DELEGATE_FIRST
            callable.delegate = new ComponentConfiguration(component)
            callable.call()

            return
        }

        throw new IllegalArgumentException("Type mismatch, expected: " + type + ", got: " + component.class)
    }

    def methodMissing(String name, args) {
        if (args != null && args.getClass().isArray()) {
            if (Array.getLength(args) == 1) {
                def clos = Array.get(args, 0)

                if (clos instanceof Closure) {
                    return component(name, clos)
                }
            }
            if (Array.getLength(args) == 2) {
                def type = Array.get(args, 0)
                def clos = Array.get(args, 1)

                if (type instanceof Class && Component.class.isAssignableFrom(type) && clos instanceof Closure) {
                    return component(name,type, clos)
                }
            }
        }

        throw new MissingMethodException("Missing method: \"$name\", args: $args")
    }
}

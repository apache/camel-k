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

import java.lang.reflect.Array

class ComponentsConfiguration {
    private final CamelContext context

    ComponentsConfiguration(CamelContext context) {
        this.context = context
    }

    def methodMissing(String name, args) {
        final Object value

        if (args == null) {
            value = null
        } else if (!args.getClass().isArray()) {
            value = args
        } else if (Array.getLength(args) == 1) {
            value = Array.get(args, 0)
        } else {
            throw new IllegalArgumentException("Unexpected argument type: " + args)
        }

        if (value instanceof Closure<?>) {
            def component = context.getComponent(name, true, false)

            value.resolveStrategy = Closure.DELEGATE_FIRST
            value.delegate = new ComponentConfiguration(component)

            return value.call()
        }

        throw new MissingMethodException("Missing method \"" + name + "\"")
    }
}

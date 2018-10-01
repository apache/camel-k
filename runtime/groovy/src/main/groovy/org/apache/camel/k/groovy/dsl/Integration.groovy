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
import org.apache.camel.builder.RouteBuilder
import org.apache.camel.k.jvm.RuntimeRegistry
import org.apache.camel.k.jvm.dsl.Components
import org.apache.camel.model.RouteDefinition
import org.apache.camel.model.rest.RestConfigurationDefinition
import org.apache.camel.model.rest.RestDefinition

class Integration {
    private final RuntimeRegistry registry

    final CamelContext context
    final Components components
    final RouteBuilder builder

    Integration(RuntimeRegistry registry, RouteBuilder builder) {
        this.registry = registry
        this.context = builder.getContext()
        this.components = new Components(this.context)
        this.builder = builder
    }

    def component(String name, Closure<?> callable) {
        def component = context.getComponent(name, true, false)

        callable.resolveStrategy = Closure.DELEGATE_ONLY
        callable.delegate = new ComponentConfiguration(component)
        callable.call()
    }

    RouteDefinition from(String endpoint) {
        return builder.from(endpoint)
    }

    RestDefinition rest() {
        return builder.rest()
    }

    def rest(Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_ONLY
        callable.delegate = builder.rest()
        callable.call()
    }

    RestConfigurationDefinition restConfiguration() {
        return builder.restConfiguration()
    }

    def restConfiguration(Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_ONLY
        callable.delegate = builder.restConfiguration()
        callable.call()
    }

    def restConfiguration(String component, Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_ONLY
        callable.delegate = builder.restConfiguration(component)
        callable.call()
    }

    def registry(Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_ONLY
        callable.delegate = registry
        callable.call()
    }
}

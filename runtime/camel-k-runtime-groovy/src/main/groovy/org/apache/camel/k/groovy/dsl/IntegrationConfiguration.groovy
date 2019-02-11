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
import org.apache.camel.Exchange
import org.apache.camel.Predicate
import org.apache.camel.Processor
import org.apache.camel.builder.RouteBuilder
import org.apache.camel.k.Runtime

import org.apache.camel.k.jvm.dsl.Components
import org.apache.camel.model.RouteDefinition

class IntegrationConfiguration {
    private final Runtime.Registry registry

    final CamelContext context
    final Components components
    final RouteBuilder builder

    IntegrationConfiguration(Runtime.Registry registry, RouteBuilder builder) {
        this.registry = registry
        this.context = builder.getContext()
        this.components = new Components(this.context)
        this.builder = builder
    }

    def context(Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_FIRST
        callable.delegate = new ContextConfiguration(context, registry)
        callable.call()
    }

    def rest(Closure<?> callable) {
        callable.resolveStrategy = Closure.DELEGATE_FIRST
        callable.delegate = new RestConfiguration(builder)
        callable.call()
    }

    RouteDefinition from(String endpoint) {
        return builder.from(endpoint)
    }

    def processor(Closure<?> callable) {
        return new Processor() {
            @Override
            void process(Exchange exchange) throws Exception {
                callable.resolveStrategy = Closure.DELEGATE_FIRST
                callable.call(exchange)
            }
        }
    }

    def predicate(Closure<?> callable) {
        return new Predicate() {
            @Override
            boolean matches(Exchange exchange) throws Exception {
                callable.resolveStrategy = Closure.DELEGATE_FIRST
                return callable.call(exchange)
            }
        }
    }
}

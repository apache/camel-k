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

import org.apache.camel.component.log.LogComponent
import org.apache.camel.component.seda.SedaComponent
import org.apache.camel.k.jvm.Runtime
import org.apache.camel.main.MainListenerSupport
import org.apache.camel.main.MainSupport
import spock.lang.Specification

import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference

class IntegrationTest extends Specification {
    def "load integration with rest"()  {
        when:
        def runtime = new Runtime()
        runtime.setDuration(5)
        runtime.load('classpath:routes-with-rest.groovy', null)
        runtime.addMainListener(new MainListenerSupport() {
            @Override
            void afterStart(MainSupport main) {
                main.stop()
            }
        })

        runtime.run()

        then:
        runtime.camelContext.restConfiguration.host == 'my-host'
        runtime.camelContext.restConfiguration.port == 9192
        runtime.camelContext.getRestConfiguration('undertow', false).host == 'my-undertow-host'
        runtime.camelContext.getRestConfiguration('undertow', false).port == 9193
    }

    def "load integration with bindings"()  {
        when:
        def runtime = new Runtime()
        runtime.setDuration(5)
        runtime.load('classpath:routes-with-bindings.groovy', null)
        runtime.addMainListener(new MainListenerSupport() {
            @Override
            void afterStart(MainSupport main) {
                main.stop()
            }
        })

        runtime.run()

        then:
        runtime.camelContext.registry.lookup('myEntry1') == 'myRegistryEntry1'
        runtime.camelContext.registry.lookup('myEntry2') == 'myRegistryEntry2'
    }

    def "load integration with component configuration"()  {
        given:
        def size = new AtomicInteger()
        def consumers = new AtomicInteger()
        def format = new AtomicReference()

        when:
        def runtime = new Runtime()
        runtime.setDuration(5)
        runtime.load('classpath:routes-with-component-configuration.groovy', null)
        runtime.addMainListener(new MainListenerSupport() {
            @Override
            void afterStart(MainSupport main) {
                def seda = runtime.camelContext.getComponent('seda', SedaComponent)
                def log = runtime.camelContext.getComponent('log', LogComponent)

                assert seda != null
                assert log != null

                size = seda.queueSize
                consumers = seda.concurrentConsumers
                format = log.exchangeFormatter

                main.stop()
            }
        })

        runtime.run()

        then:
        size == 1234
        consumers == 12
        format != null
    }
}

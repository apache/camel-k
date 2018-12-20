/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.apache.camel.k.kotlin

import org.apache.camel.impl.DefaultCamelContext
import org.apache.camel.k.InMemoryRegistry
import org.apache.camel.k.Source
import org.apache.camel.k.support.RuntimeSupport
import org.apache.camel.model.ProcessDefinition
import org.apache.camel.model.ToDefinition
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test

class LoaderTest {

    @Test
    fun `load route from classpath`() {
        var source = Source.create("classpath:routes.kts")
        val loader = RuntimeSupport.loaderFor(DefaultCamelContext(), source)
        val builder = loader.load(InMemoryRegistry(), source)

        assertThat(loader).isInstanceOf(KotlinRoutesLoader::class.java)
        assertThat(builder).isNotNull

        builder.configure()

        val routes = builder.routeCollection.routes
        assertThat(routes).hasSize(1)
        assertThat(routes[0].inputs[0].endpointUri).isEqualTo("timer:tick")
        assertThat(routes[0].outputs[0]).isInstanceOf(ProcessDefinition::class.java)
        assertThat(routes[0].outputs[1]).isInstanceOf(ToDefinition::class.java)
    }
}

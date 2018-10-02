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

import org.apache.camel.builder.RouteBuilder
import org.apache.camel.k.jvm.Language
import org.apache.camel.k.jvm.RoutesLoader
import org.apache.camel.k.jvm.RuntimeRegistry
import org.apache.camel.util.ResourceHelper
import java.io.InputStreamReader
import javax.script.ScriptEngineManager
import javax.script.SimpleBindings

class KotlinRoutesLoader : RoutesLoader {

    override fun getSupportedLanguages(): List<Language> {
        return listOf(Language.Kotlin)
    }

    @Throws(Exception::class)
    override fun load(registry: RuntimeRegistry, resource: String): RouteBuilder {
        return object : RouteBuilder() {
            @Throws(Exception::class)
            override fun configure() {
                val context = context
                val manager = ScriptEngineManager()
                val engine = manager.getEngineByExtension("kts")
                val bindings = SimpleBindings()

                bindings["builder"] = this
                bindings["registry"] = registry
                bindings["context"] = context

                ResourceHelper.resolveMandatoryResourceAsInputStream(context, resource).use { `is` ->
                    val pre = """
                        val builder = bindings["builder"] as org.apache.camel.builder.RouteBuilder

                        fun rest(block: org.apache.camel.model.rest.RestDefinition.() -> Unit) {
                            val delegate = builder.rest()
                            delegate.block()
                        }

                        fun restConfiguration(block: org.apache.camel.model.rest.RestConfigurationDefinition.() -> Unit) {
                            val delegate = builder.restConfiguration()
                            delegate.block()
                        }

                        fun restConfiguration(component: String, block: org.apache.camel.model.rest.RestConfigurationDefinition.() -> Unit) {
                            val delegate = builder.restConfiguration(component)
                            delegate.block()
                        }

                        fun context(block: org.apache.camel.k.kotlin.dsl.ContextConfiguration.() -> Unit) {
                            val delegate = org.apache.camel.k.kotlin.dsl.ContextConfiguration(
                                context  = bindings["context"] as org.apache.camel.CamelContext,
                                registry = bindings["registry"] as org.apache.camel.k.jvm.RuntimeRegistry
                            )

                            delegate.block()
                        }

                        fun from(uri: String): org.apache.camel.model.RouteDefinition = builder.from(uri)
                    """.trimIndent()

                    engine.eval(pre, bindings)
                    engine.eval(InputStreamReader(`is`), bindings)
                }
            }
        }
    }
}

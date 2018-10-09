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
import org.apache.camel.k.kotlin.dsl.IntegrationConfiguration
import org.apache.camel.util.ResourceHelper
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import java.io.File
import java.io.InputStreamReader
import kotlin.script.experimental.api.*
import kotlin.script.experimental.host.toScriptSource
import kotlin.script.experimental.jvm.dependenciesFromClassloader
import kotlin.script.experimental.jvm.javaHome
import kotlin.script.experimental.jvm.jvm
import kotlin.script.experimental.jvmhost.BasicJvmScriptEvaluator
import kotlin.script.experimental.jvmhost.BasicJvmScriptingHost
import kotlin.script.experimental.jvmhost.JvmScriptCompiler

class KotlinRoutesLoader : RoutesLoader {
    companion object {
        val LOGGER : Logger = LoggerFactory.getLogger(KotlinRoutesLoader::class.java)
    }

    override fun getSupportedLanguages(): List<Language> {
        return listOf(Language.Kotlin)
    }

    @Throws(Exception::class)
    override fun load(registry: RuntimeRegistry, resource: String): RouteBuilder {
        return object : RouteBuilder() {
            @Throws(Exception::class)
            override fun configure() {
                val builder = this
                val compiler = JvmScriptCompiler()
                val evaluator = BasicJvmScriptEvaluator()
                val host = BasicJvmScriptingHost(compiler = compiler, evaluator = evaluator)
                val javaHome = System.getenv("KOTLIN_JDK_HOME") ?: "/usr/lib/jvm/java"

                LOGGER.info("JAVA_HOME is set to {}", javaHome)

                ResourceHelper.resolveMandatoryResourceAsInputStream(context, resource).use { `is` ->
                    val result = host.eval(
                        InputStreamReader(`is`).readText().toScriptSource(),
                        ScriptCompilationConfiguration {
                            baseClass(IntegrationConfiguration::class)
                            jvm {
                                //
                                // This is needed as workaround for:
                                //     https://youtrack.jetbrains.com/issue/KT-27497
                                //
                                javaHome(File(javaHome))

                                //
                                // The Kotlin script compiler does not inherit
                                // the classpath by default
                                //
                                dependenciesFromClassloader(wholeClasspath = true)
                            }
                        },
                        ScriptEvaluationConfiguration {
                            //
                            // Arguments used to initialize the script base class
                            //
                            constructorArgs(registry, builder)
                        }
                    )

                    for (report in result.reports) {
                        if (report.severity == ScriptDiagnostic.Severity.ERROR) {
                            LOGGER.error("{}", report.message, report.exception)
                        } else if (report.severity == ScriptDiagnostic.Severity.WARNING) {
                            LOGGER.warn("{}", report.message, report.exception)
                        } else {
                            LOGGER.info("{}", report.message)
                        }
                    }
                }
            }
        }
    }
}

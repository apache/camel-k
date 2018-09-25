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
package org.apache.camel.k.kotlin;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.Collections;
import java.util.List;
import javax.script.Bindings;
import javax.script.ScriptEngine;
import javax.script.ScriptEngineManager;
import javax.script.SimpleBindings;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.dsl.Components;
import org.apache.camel.k.jvm.Language;
import org.apache.camel.k.jvm.RoutesLoader;
import org.apache.camel.util.ResourceHelper;

public class KotlinRoutesLoader implements RoutesLoader {

    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.Kotlin);
    }

    @Override
    public RouteBuilder load(String resource) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                final CamelContext context = getContext();
                final ScriptEngineManager manager = new ScriptEngineManager();
                final ScriptEngine engine = manager.getEngineByExtension("kts");
                final Bindings bindings = new SimpleBindings();

                // Exposed to the underlying script, but maybe better to have
                // a nice dsl
                bindings.put("builder", this);
                bindings.put("context", context);
                bindings.put("components", new Components(context));

                try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(context, resource)) {
                    StringBuilder builder = new StringBuilder();

                    // extract global objects from 'bindings'
                    builder.append("val builder = bindings[\"builder\"] as org.apache.camel.builder.RouteBuilder").append('\n');
                    builder.append("val context = bindings[\"context\"] as org.apache.camel.CamelContext").append('\n');
                    builder.append("val components = bindings[\"components\"] as org.apache.camel.k.jvm.dsl.Components").append('\n');

                    // create aliases for common functions
                    builder.append("fun from(uri: String): org.apache.camel.model.RouteDefinition = builder.from(uri)").append('\n');
                    builder.append("fun rest(): org.apache.camel.model.rest.RestDefinition = builder.rest()").append('\n');
                    builder.append("fun restConfiguration(): org.apache.camel.model.rest.RestConfigurationDefinition = builder.restConfiguration()").append('\n');

                    engine.eval(builder.toString(), bindings);
                    engine.eval(new InputStreamReader(is), bindings);
                }
            }
        };
    }
}

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
package org.apache.camel.k.jvm.loader;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.Collections;
import java.util.List;
import java.util.function.Function;
import java.util.function.Supplier;
import javax.script.Bindings;
import javax.script.ScriptEngine;
import javax.script.ScriptEngineManager;
import javax.script.SimpleBindings;

import org.apache.camel.CamelContext;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.Source;
import org.apache.camel.k.jvm.dsl.Components;
import org.apache.camel.k.support.URIResolver;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.rest.RestConfigurationDefinition;
import org.apache.camel.model.rest.RestDefinition;

public class JavaScriptLoader implements RoutesLoader {
    @Override
    public List<String> getSupportedLanguages() {
        return Collections.singletonList("js");
    }

    @Override
    public RouteBuilder load(RuntimeRegistry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                final CamelContext context = getContext();
                final ScriptEngineManager manager = new ScriptEngineManager();
                final ScriptEngine engine = manager.getEngineByName("nashorn");
                final Bindings bindings = new SimpleBindings();

                // Exposed to the underlying script, but maybe better to have
                // a nice dsl
                bindings.put("builder", this);
                bindings.put("context", context);
                bindings.put("components", new Components(context));
                bindings.put("registry", registry);
                bindings.put("from", (Function<String, RouteDefinition>) uri -> from(uri));
                bindings.put("rest", (Supplier<RestDefinition>) () -> rest());
                bindings.put("restConfiguration", (Supplier<RestConfigurationDefinition>) () -> restConfiguration());

                try (InputStream is = URIResolver.resolve(context, source)) {
                    engine.eval(new InputStreamReader(is), bindings);
                }
            }
        };
    }
}

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
package org.apache.camel.k.groovy;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.util.Collections;
import java.util.List;

import groovy.lang.Binding;
import groovy.lang.GroovyShell;
import groovy.util.DelegatingScript;
import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.Language;
import org.apache.camel.k.jvm.RoutesLoader;
import org.apache.camel.k.jvm.dsl.Scripting;
import org.apache.camel.util.ResourceHelper;
import org.codehaus.groovy.control.CompilerConfiguration;

public class GroovyRoutesLoader implements RoutesLoader {
    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.Groovy);
    }

    @Override
    public RouteBuilder load(String resource) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                CompilerConfiguration cc = new CompilerConfiguration();
                cc.setScriptBaseClass(DelegatingScript.class.getName());

                ClassLoader cl = Thread.currentThread().getContextClassLoader();
                GroovyShell sh = new GroovyShell(cl, new Binding(), cc);

                try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(getContext(), resource)) {
                    Reader reader = new InputStreamReader(is);
                    DelegatingScript script = (DelegatingScript) sh.parse(reader);

                    // set the delegate target
                    script.setDelegate(new Scripting(this));
                    script.run();
                }
            }
        };
    }
}

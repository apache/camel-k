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

import org.apache.camel.CamelContext;
import org.apache.camel.k.jvm.Runtime;
import org.apache.camel.main.MainListenerSupport;
import org.apache.camel.main.MainSupport;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class RuntimeRegistryTest {
    @Test
    public void testLoadRouteWithBindings() throws Exception {
        Runtime runtime = new Runtime();
        runtime.setDuration(5);
        runtime.load("classpath:routes-with-bindings.kts", null);
        runtime.addMainListener(new MainListenerSupport() {
            @Override
            public void afterStart(MainSupport main) {
                try {
                    CamelContext context = main.getCamelContexts().get(0);
                    Object value = context.getRegistry().lookup("myEntry");

                    assertThat(value).isEqualTo("myRegistryEntry");

                    main.stop();
                } catch (Exception e) {
                    throw new RuntimeException(e);
                }
            }
        });

        runtime.run();
    }
}

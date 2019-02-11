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
package org.apache.camel.k.jvm;

import java.util.Properties;
import java.util.concurrent.ThreadLocalRandom;

import org.apache.camel.CamelContext;
import org.apache.camel.component.seda.SedaComponent;
import org.apache.camel.k.Constants;
import org.apache.camel.k.ContextCustomizer;
import org.apache.camel.k.Runtime;
import org.apache.camel.k.listener.ContextConfigurer;
import org.apache.camel.k.listener.ContextLifecycleConfigurer;
import org.apache.camel.k.support.RuntimeSupport;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class PropertiesTest {

    @Test
    public void testLoadProperties() throws Exception {
        Properties properties = RuntimeSupport.loadProperties("src/test/resources/conf.properties", "src/test/resources/conf.d");

        ApplicationRuntime runtime = new ApplicationRuntime();
        runtime.setProperties(properties);
        runtime.addListener(new ContextConfigurer());
        runtime.addListener(new ContextLifecycleConfigurer());
        runtime.addListener(Runtime.Phase.Started, r -> {
            CamelContext context = r.getContext();
            assertThat(context.resolvePropertyPlaceholders("{{root.key}}")).isEqualTo("root.value");
            assertThat(context.resolvePropertyPlaceholders("{{001.key}}")).isEqualTo("001.value");
            assertThat(context.resolvePropertyPlaceholders("{{002.key}}")).isEqualTo("002.value");
            assertThat(context.resolvePropertyPlaceholders("{{a.key}}")).isEqualTo("a.002");
            runtime.stop();
        });

        runtime.run();
    }

    @Test
    public void testSystemProperties() throws Exception {
        System.setProperty("my.property", "my.value");

        try {
            ApplicationRuntime runtime = new ApplicationRuntime();
            runtime.setProperties(System.getProperties());
            runtime.addListener(new ContextConfigurer());
            runtime.addListener(new ContextLifecycleConfigurer());
            runtime.addListener(Runtime.Phase.Started, r -> {
                CamelContext context = r.getContext();
                String value = context.resolvePropertyPlaceholders("{{my.property}}");

                assertThat(value).isEqualTo("my.value");
                runtime.stop();
            });

            runtime.run();
        } finally {
            System.getProperties().remove("my.property");
        }
    }

    @Test
    public void testComponentConfiguration() throws Exception {
        int queueSize1 = ThreadLocalRandom.current().nextInt(10, 100);
        int queueSize2 = ThreadLocalRandom.current().nextInt(10, 100);

        System.setProperty("camel.component.seda.queueSize", Integer.toString(queueSize1));
        System.setProperty("camel.component.my-seda.queueSize", Integer.toString(queueSize2));

        try {
            ApplicationRuntime runtime = new ApplicationRuntime();
            runtime.setProperties(System.getProperties());
            runtime.getRegistry().bind("my-seda", new SedaComponent());
            runtime.addListener(new ContextConfigurer());
            runtime.addListener(new ContextLifecycleConfigurer());
            runtime.addListener(Runtime.Phase.Started, r -> {
                CamelContext context = r.getContext();
                assertThat(context.getComponent("seda", true)).hasFieldOrPropertyWithValue("queueSize", queueSize1);
                assertThat(context.getComponent("my-seda", true)).hasFieldOrPropertyWithValue("queueSize", queueSize2);
                runtime.stop();
            });

            runtime.run();
        } finally {
            System.getProperties().remove("camel.component.seda.queueSize");
            System.getProperties().remove("camel.component.my-seda.queueSize");
        }
    }

    @Test
    public void testContextConfiguration() throws Exception {
        System.setProperty("camel.context.messageHistory", "false");
        System.setProperty("camel.context.loadTypeConverters", "false");

        try {
            ApplicationRuntime runtime = new ApplicationRuntime();
            runtime.setProperties(System.getProperties());
            runtime.addListener(new ContextConfigurer());
            runtime.addListener(new ContextLifecycleConfigurer());
            runtime.addListener(Runtime.Phase.Started, r -> {
                CamelContext context = r.getContext();
                assertThat(context.isMessageHistory()).isFalse();
                assertThat(context.isLoadTypeConverters()).isFalse();
                runtime.stop();
            });

            runtime.run();
        } finally {
            System.getProperties().remove("camel.context.messageHistory");
            System.getProperties().remove("camel.context.loadTypeConverters");
        }
    }

    @Test
    public void testContextCustomizerFromProperty() throws Exception {
        System.setProperty(Constants.PROPERTY_CAMEL_K_CUSTOMIZER, "test");
        System.setProperty("customizer.test.messageHistory", "false");

        ApplicationRuntime runtime = new ApplicationRuntime();
        runtime.setProperties(System.getProperties());
        runtime.addListener(new ContextConfigurer());
        runtime.addListener(new ContextLifecycleConfigurer());
        runtime.addListener(Runtime.Phase.Started, r -> {
            CamelContext context = r.getContext();
            assertThat(context.isMessageHistory()).isFalse();
            assertThat(context.isLoadTypeConverters()).isFalse();
            runtime.stop();
        });

        runtime.run();
    }

    @Test
    public void testContextCustomizerFromRegistry() throws Exception {
        ApplicationRuntime runtime = new ApplicationRuntime();
        runtime.setProperties(System.getProperties());
        runtime.addListener(new ContextConfigurer());
        runtime.addListener(new ContextLifecycleConfigurer());
        runtime.getRegistry().bind("c1", (ContextCustomizer) (camelContext, registry) -> {
            camelContext.setMessageHistory(false);
            camelContext.setLoadTypeConverters(false);
        });
        runtime.addListener(Runtime.Phase.Started, r -> {
            CamelContext context = r.getContext();
            assertThat(context.isMessageHistory()).isFalse();
            assertThat(context.isLoadTypeConverters()).isFalse();
            runtime.stop();
        });

        runtime.run();
    }
}

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

import java.util.concurrent.ThreadLocalRandom;

import org.apache.camel.CamelContext;
import org.apache.camel.component.seda.SedaComponent;
import org.apache.camel.main.Main;
import org.apache.camel.main.MainListenerSupport;
import org.apache.camel.main.MainSupport;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class PropertiesTest {

    @Test
    public void testSystemProperties() throws Exception {
        System.setProperty("my.property", "my.value");

        try {
            Main main = new Main();
            main.setDuration(5);
            main.addMainListener(new Application.ComponentPropertiesBinder());
            main.addMainListener(new MainListenerSupport() {
                @Override
                public void afterStart(MainSupport main) {
                    try {
                        CamelContext context = main.getCamelContexts().get(0);
                        String value = context.resolvePropertyPlaceholders("{{my.property}}");

                        assertThat(value).isEqualTo("my.value");

                        main.stop();
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                }
            });

            main.run();
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
            Main main = new Main();
            main.setDuration(5);
            main.bind("my-seda", new SedaComponent());
            main.addMainListener(new Application.ComponentPropertiesBinder());
            main.addMainListener(new MainListenerSupport() {
                @Override
                public void afterStart(MainSupport main) {
                    try {
                        CamelContext context = main.getCamelContexts().get(0);

                        assertThat(context.getComponent("seda", true)).hasFieldOrPropertyWithValue("queueSize", queueSize1);
                        assertThat(context.getComponent("my-seda", true)).hasFieldOrPropertyWithValue("queueSize", queueSize2);

                        main.stop();
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                }
            });

            main.run();
        } finally {
            System.getProperties().remove("camel.component.seda.queueSize");
        }
    }
}

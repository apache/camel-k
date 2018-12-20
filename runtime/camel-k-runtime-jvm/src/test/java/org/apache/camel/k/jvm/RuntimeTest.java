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

import java.io.InputStream;
import java.nio.charset.Charset;
import java.util.List;

import org.apache.camel.CamelContext;
import org.apache.camel.Route;
import org.apache.camel.util.ObjectHelper;
import org.apache.camel.util.ResourceHelper;
import org.apache.commons.io.IOUtils;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Java6Assertions.assertThat;

public class RuntimeTest {

    @Test
    void testLoadMultipleRoutes() throws Exception {
        Runtime runtime = new Runtime();
        runtime.load(new String[]{
            "classpath:r1.js",
            "classpath:r2.mytype?language=js",
        });

        try {
            runtime.start();

            CamelContext context = runtime.getCamelContext();
            List<Route> routes = context.getRoutes();

            assertThat(routes).hasSize(2);
            assertThat(routes).anyMatch(p -> ObjectHelper.equal("r1", p.getId()));
            assertThat(routes).anyMatch(p -> ObjectHelper.equal("r2", p.getId()));
        } finally {
            runtime.stop();
        }
    }


    @Test
    void testLoadResource() throws Exception {
        ApplicationSupport.configureStreamHandler();

        CamelContext context = new Runtime().getCamelContext();

        try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(context, "platform:my-resource.txt")) {
            String content = IOUtils.toString(is, Charset.defaultCharset());

            assertThat(content).isEqualTo("value from file resource");
        }

        try (InputStream is = ResourceHelper.resolveMandatoryResourceAsInputStream(context, "platform:my-other-resource.txt")) {
            String content = IOUtils.toString(is, Charset.defaultCharset());

            assertThat(content).isEqualTo("value from env");
        }
    }
}

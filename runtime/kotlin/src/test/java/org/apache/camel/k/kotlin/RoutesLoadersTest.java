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

import java.util.List;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.jvm.RoutesLoader;
import org.apache.camel.k.jvm.RoutesLoaders;
import org.apache.camel.model.ProcessDefinition;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.ToDefinition;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class RoutesLoadersTest {

    @Test
    public void testLoadKotlin() throws Exception {
        String resource = "classpath:routes.kts";
        RoutesLoader loader = RoutesLoaders.loaderFor(resource, null);
        RouteBuilder builder = loader.load(resource);

        assertThat(loader).isInstanceOf(KotlinRoutesLoader.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ProcessDefinition.class);
        assertThat(routes.get(0).getOutputs().get(1)).isInstanceOf(ToDefinition.class);
    }
}

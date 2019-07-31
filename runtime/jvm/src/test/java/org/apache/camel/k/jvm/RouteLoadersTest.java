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

import java.util.List;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.ToDefinition;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;

public class RouteLoadersTest {

    @Test
    public void testLoadClass() throws Exception {
        String resource = "classpath:" + MyRoutes.class.getCanonicalName();
        RoutesLoader loader = RouteLoaders.loaderFor(resource);
        RouteBuilder builder = loader.load(resource);

        assertThat(loader).isSameAs(RouteLoaders.JavaClass);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJava() throws Exception {
        String resource = "classpath:MyRoutes.java";
        RoutesLoader loader = RouteLoaders.loaderFor(resource);
        RouteBuilder builder = loader.load(resource);

        assertThat(loader).isSameAs(RouteLoaders.JavaSource);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJavaScript() throws Exception {
        String resource = "classpath:routes.js";
        RoutesLoader loader = RouteLoaders.loaderFor(resource);
        RouteBuilder builder = loader.load(resource);

        assertThat(loader).isSameAs(RouteLoaders.JavaScript);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadGroovy() throws Exception {
        String resource = "classpath:routes.groovy";
        RoutesLoader loader = RouteLoaders.loaderFor(resource);
        RouteBuilder builder = loader.load(resource);

        assertThat(loader).isSameAs(RouteLoaders.Groovy);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testResourceWithoutScheme() {
        RouteLoaders.loaderFor("routes.js");
    }

    @Test(expected = IllegalArgumentException.class)
    public void testResourceWithIllegalScheme() {
        RouteLoaders.loaderFor("http:routes.js");
    }
}

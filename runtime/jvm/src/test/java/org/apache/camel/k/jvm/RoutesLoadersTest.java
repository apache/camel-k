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
import org.apache.camel.model.ProcessDefinition;
import org.apache.camel.model.RouteDefinition;
import org.apache.camel.model.SetBodyDefinition;
import org.apache.camel.model.ToDefinition;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.AssertionsForClassTypes.assertThatExceptionOfType;

public class RoutesLoadersTest {

    @Test
    public void testLoadClass() throws Exception {
        Source source = Source.create("classpath:" + MyRoutes.class.getName() + ".class");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaClass.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJava() throws Exception {
        Source source = Source.create("classpath:MyRoutes.java");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaSource.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJavaWithNestedClass() throws Exception {
        Source source = Source.create("classpath:MyRoutesWithNestedClass.java");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaSource.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(SetBodyDefinition.class);
        assertThat(routes.get(0).getOutputs().get(1)).isInstanceOf(ProcessDefinition.class);
        assertThat(routes.get(0).getOutputs().get(2)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJavaScript() throws Exception {
        Source source = Source.create("classpath:routes.js");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaScript.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadCompressedRoute() throws Exception {
        Source source = Source.create("classpath:routes-compressed.js.gz.b64?language=js&compression=true");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaScript.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadJavaScriptWithCustomExtension() throws Exception {
        Source source = Source.create("classpath:routes.mytype?language=js");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.JavaScript.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testLoadXml() throws Exception {
        Source source = Source.create("classpath:routes.xml");
        RoutesLoader loader = RoutesLoaders.loaderFor(source);
        RouteBuilder builder = loader.load(new SimpleRuntimeRegistry(), source);

        assertThat(loader).isInstanceOf(RoutesLoaders.Xml.class);
        assertThat(builder).isNotNull();

        builder.configure();

        List<RouteDefinition> routes = builder.getRouteCollection().getRoutes();
        assertThat(routes).hasSize(1);
        assertThat(routes.get(0).getInputs().get(0).getEndpointUri()).isEqualTo("timer:tick");
        assertThat(routes.get(0).getOutputs().get(0)).isInstanceOf(ToDefinition.class);
    }

    @Test
    public void testResourceWithoutScheme() {
        assertThatExceptionOfType(IllegalArgumentException.class).isThrownBy(
            () -> Source.create("routes.js")
        );
    }

    @Test
    public void testResourceWithIllegalScheme() {
        assertThatExceptionOfType(IllegalArgumentException.class).isThrownBy(
            () -> Source.create("http:routes.js")
        );
    }

    @Test
    public void testUnsupportedLanguage() {
        assertThatExceptionOfType(IllegalArgumentException.class).isThrownBy(
            () -> Source.create("  test")
        );
    }

    public static class MyRoutes extends RouteBuilder {
        @Override
        public void configure() throws Exception {
            from("timer:tick")
                .to("log:info");
        }
    }
}

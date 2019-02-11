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
package org.apache.camel.k.spring.boot;

import java.util.Properties;

import org.apache.camel.k.support.PlatformStreamHandler;
import org.apache.camel.k.support.RuntimeSupport;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.context.support.PropertySourcesPlaceholderConfigurer;

@SpringBootApplication
public class Application {
    static {
        //
        // Install a custom protocol handler to support discovering resources
        // from the platform i.e. in knative, resources are provided through
        // env var as it is not possible to mount config maps / secrets.
        //
        // TODO: we should remove as soon as we get a knative version that
        //       includes https://github.com/knative/serving/pull/3061
        //
        PlatformStreamHandler.configure();
    }

    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }

    @Bean
    public static PropertySourcesPlaceholderConfigurer propertySourcesPlaceholderConfigurer() {
        // load properties using default behaviour
        final Properties properties = RuntimeSupport.loadProperties();

        // set spring boot specific properties
        properties.put("camel.springboot.main-run-controller", "true");
        properties.put("camel.springboot.name", "camel-k");
        properties.put("camel.springboot.streamCachingEnabled", "true");
        properties.put("camel.springboot.xml-routes", "false");
        properties.put("camel.springboot.xml-rests", "false");
        properties.put("camel.springboot.jmx-enabled", "false");

        // set loaded properties as default properties
        PropertySourcesPlaceholderConfigurer configurer = new PropertySourcesPlaceholderConfigurer();
        configurer.setProperties(properties);

        return configurer;
    }
}

/*
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


package customizers;

// camel-k: language=java

import org.apache.camel.BindToRegistry;
import org.apache.camel.CamelContext;
import org.apache.camel.PropertyInject;
import org.apache.camel.opentracing.OpenTracingTracer;

import io.opentracing.Tracer;

import io.jaegertracing.Configuration;
import io.jaegertracing.Configuration.ReporterConfiguration;
import io.jaegertracing.Configuration.SamplerConfiguration;
import io.jaegertracing.Configuration.SenderConfiguration;

public class OpentracingCustomizer {

    @BindToRegistry
    public static OpenTracingTracer tracer(
        CamelContext ctx, 
        @PropertyInject("env:CAMEL_K_INTEGRATION") String name, 
        @PropertyInject("jaeger.endpoint") String endpoint) {

            OpenTracingTracer openTracingTracer = new OpenTracingTracer();
            openTracingTracer.setTracer(new Configuration(name)
                .withReporter(new ReporterConfiguration()
                    .withSender(new SenderConfiguration()
                        .withEndpoint(endpoint)
                    )
                )
                .withSampler(new SamplerConfiguration()
                    .withType("const")    
                    .withParam(1)
                )
                .getTracer()
            );
            openTracingTracer.init(ctx);
            return openTracingTracer;
    }

}
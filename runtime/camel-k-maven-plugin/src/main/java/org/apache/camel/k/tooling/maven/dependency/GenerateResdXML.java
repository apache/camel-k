package org.apache.camel.k.tooling.maven.dependency;
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

import java.io.PrintWriter;
import java.io.Writer;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

import io.swagger.models.Swagger;
import io.swagger.parser.SwaggerParser;
import org.apache.camel.CamelContext;
import org.apache.camel.generator.swagger.RestDslXmlGenerator;
import org.apache.camel.impl.DefaultCamelContext;
import org.apache.maven.plugin.AbstractMojo;
import org.apache.maven.plugin.MojoExecutionException;
import org.apache.maven.plugins.annotations.LifecyclePhase;
import org.apache.maven.plugins.annotations.Mojo;
import org.apache.maven.plugins.annotations.Parameter;
import org.apache.maven.plugins.annotations.ResolutionScope;

@Mojo(
    name = "generate-rest-xml",
    inheritByDefault = false,
    defaultPhase = LifecyclePhase.GENERATE_SOURCES,
    requiresDependencyResolution = ResolutionScope.COMPILE,
    threadSafe = true,
    requiresProject = false)
public class GenerateResdXML extends AbstractMojo {
    @Parameter(property = "openapi.spec")
    private String inputFile;
    @Parameter(property = "dsl.out")
    private String outputFile;

    @Override
    public void execute() throws MojoExecutionException {
        if (inputFile == null) {
            throw new MojoExecutionException("Missing input file: " + inputFile);
        }

        Path input = Paths.get(this.inputFile);
        if (!Files.exists(input)) {
            throw new MojoExecutionException("Unable to read the input file: " + inputFile);
        }

        final SwaggerParser sparser = new SwaggerParser();
        final Swagger swagger = sparser.read(inputFile);
        if (swagger == null) {
            throw new MojoExecutionException("Unable to read the swagger file: " + inputFile);
        }

        try {
            final Writer writer;

            if (outputFile != null) {
                Path output = Paths.get(this.outputFile);

                if (output.getParent() != null && Files.notExists(output.getParent())) {
                    Files.createDirectories(output.getParent());
                }
                if (Files.exists(output)) {
                    Files.delete(output);
                }

                writer = Files.newBufferedWriter(output);
            } else {
                writer = new PrintWriter(System.out);
            }

            final CamelContext context = new DefaultCamelContext();
            final String dsl = RestDslXmlGenerator.toXml(swagger).generate(context);

            writer.write(dsl);
            writer.close();
        } catch (Exception e) {
            throw new MojoExecutionException("Exception while generating rest xml", e);
        }
    }
}

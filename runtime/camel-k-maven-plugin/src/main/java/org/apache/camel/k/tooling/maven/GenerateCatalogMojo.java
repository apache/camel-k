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
package org.apache.camel.k.tooling.maven;

import java.io.IOException;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Comparator;
import java.util.Map;
import java.util.Objects;
import java.util.ServiceLoader;
import java.util.SortedMap;
import java.util.TreeMap;
import java.util.stream.StreamSupport;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.dataformat.yaml.YAMLGenerator;
import org.apache.camel.catalog.DefaultCamelCatalog;
import org.apache.camel.k.tooling.maven.model.CamelArtifact;
import org.apache.camel.k.tooling.maven.model.CatalogComponentDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogDataFormatDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogLanguageDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogProcessor;
import org.apache.camel.k.tooling.maven.model.CatalogSupport;
import org.apache.camel.k.tooling.maven.model.crd.CamelCatalog;
import org.apache.camel.k.tooling.maven.model.crd.CamelCatalogSpec;
import org.apache.camel.k.tooling.maven.model.k8s.ObjectMeta;
import org.apache.commons.lang3.StringUtils;
import org.apache.maven.plugin.AbstractMojo;
import org.apache.maven.plugin.MojoExecutionException;
import org.apache.maven.plugin.MojoFailureException;
import org.apache.maven.plugins.annotations.LifecyclePhase;
import org.apache.maven.plugins.annotations.Mojo;
import org.apache.maven.plugins.annotations.Parameter;
import org.apache.maven.plugins.annotations.ResolutionScope;
import org.apache.maven.project.MavenProject;

@Mojo(
    name = "generate-catalog",
    defaultPhase = LifecyclePhase.GENERATE_RESOURCES,
    threadSafe = true,
    requiresDependencyResolution = ResolutionScope.COMPILE_PLUS_RUNTIME,
    requiresDependencyCollection = ResolutionScope.COMPILE_PLUS_RUNTIME)
public class GenerateCatalogMojo extends AbstractMojo {

    @Parameter(readonly = true, defaultValue = "${project}")
    private MavenProject project;

    @Parameter(property = "catalog.path", defaultValue = "${project.build.directory}")
    private String outputPath;

    @Parameter(property = "catalog.file", defaultValue = "camel-catalog-${catalog.version}.yaml")
    private String outputFile;

    // ********************
    //
    // ********************

    @Override
    public void execute() throws MojoExecutionException, MojoFailureException {
        final Path output = Paths.get(this.outputPath, this.outputFile);

        try {
            if (Files.notExists(output.getParent())) {
                Files.createDirectories(output.getParent());
            }

            if (Files.exists(output)) {
                Files.delete(output);
            }
        } catch (IOException e) {
            throw new MojoExecutionException("Exception while generating camel catalog", e);
        }

        final SortedMap<String, CamelArtifact> artifacts = new TreeMap<>();
        final org.apache.camel.catalog.CamelCatalog catalog = new DefaultCamelCatalog();

        try {
            processComponents(catalog, artifacts);
            processLanguages(catalog, artifacts);
            processDataFormats(catalog, artifacts);

            ServiceLoader<CatalogProcessor> processors = ServiceLoader.load(CatalogProcessor.class);
            Comparator<CatalogProcessor> comparator = Comparator.comparingInt(CatalogProcessor::getOrder);

            //
            // post process catalog
            //
            StreamSupport.stream(processors.spliterator(), false).sorted(comparator).filter(p -> p.accepts(catalog)).forEach(p -> {
                getLog().info("Executing processor: " + p.getClass().getName());

                p.process(project, catalog, artifacts);
            });

            //
            // apiVersion: camel.apache.org/v1alpha1
            // kind: CamelCatalog
            // metadata:
            //   name: catalog-x.y.z
            //   labels:
            //     app: "camel-k"
            //     camel.apache.org/catalog.version: x.y.x
            //     camel.apache.org/catalog.loader.version: x.y.z
            // spec:
            //   version:
            // status:
            //   artifacts:
            //
            try (Writer writer = Files.newBufferedWriter(output, StandardCharsets.UTF_8)) {
                CamelCatalog cr = new CamelCatalog.Builder()
                    .metadata(new ObjectMeta.Builder()
                        .name("camel-catalog-" + catalog.getCatalogVersion())
                        .putLabels("app", "camel-k")
                        .putLabels("camel.apache.org/catalog.version", catalog.getCatalogVersion())
                        .putLabels("camel.apache.org/catalog.loader.version", catalog.getLoadedVersion())
                        .build())
                    .spec(new CamelCatalogSpec.Builder()
                        .version(catalog.getCatalogVersion())
                        .artifacts(artifacts)
                        .build())
                    .build();

                YAMLFactory factory = new YAMLFactory()
                    .configure(YAMLGenerator.Feature.MINIMIZE_QUOTES, true)
                    .configure(YAMLGenerator.Feature.ALWAYS_QUOTE_NUMBERS_AS_STRINGS, true)
                    .configure(YAMLGenerator.Feature.USE_NATIVE_TYPE_ID, false)
                    .configure(YAMLGenerator.Feature.WRITE_DOC_START_MARKER, false);

                //new Yaml(representer, options).dump(cr, writer);
                ObjectMapper mapper = new ObjectMapper(factory);
                mapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
                mapper.setSerializationInclusion(JsonInclude.Include.NON_EMPTY);
                mapper.writeValue(writer, cr);
            }
        } catch (IOException e) {
            throw new MojoExecutionException("Exception while generating catalog", e);
        }
    }

    private void processComponents(org.apache.camel.catalog.CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
        for (String name : catalog.findComponentNames()) {
            String json = catalog.componentJSonSchema(name);
            CatalogComponentDefinition definition = CatalogSupport.unmarshallComponent(json);

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                }

                definition.getSchemes()
                    .map(StringUtils::trimToNull)
                    .filter(Objects::nonNull)
                    .forEach(artifact::createScheme);

                return artifact;
            });
        }
    }

    private void processLanguages(org.apache.camel.catalog.CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
        for (String name : catalog.findLanguageNames()) {
            String json = catalog.languageJSonSchema(name);
            CatalogLanguageDefinition definition = CatalogSupport.unmarshallLanguage(json);

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                }

                artifact.addLanguage(definition.getName());

                return artifact;
            });
        }
    }

    private void processDataFormats(org.apache.camel.catalog.CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
        for (String name : catalog.findDataFormatNames()) {
            String json = catalog.dataFormatJSonSchema(name);
            CatalogDataFormatDefinition definition = CatalogSupport.unmarshallDataFormat(json);

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                }

                artifact.addDataformats(definition.getName());

                return artifact;
            });
        }
    }
}

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
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.ServiceLoader;
import java.util.TreeMap;
import java.util.stream.StreamSupport;

import org.apache.camel.catalog.CamelCatalog;
import org.apache.camel.catalog.DefaultCamelCatalog;
import org.apache.camel.k.tooling.maven.model.CamelArtifact;
import org.apache.camel.k.tooling.maven.model.CatalogComponentDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogDataFormatDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogLanguageDefinition;
import org.apache.camel.k.tooling.maven.model.CatalogProcessor;
import org.apache.camel.k.tooling.maven.model.CatalogSupport;
import org.apache.commons.lang3.StringUtils;
import org.apache.maven.plugin.AbstractMojo;
import org.apache.maven.plugin.MojoExecutionException;
import org.apache.maven.plugin.MojoFailureException;
import org.apache.maven.plugins.annotations.LifecyclePhase;
import org.apache.maven.plugins.annotations.Mojo;
import org.apache.maven.plugins.annotations.Parameter;
import org.apache.maven.plugins.annotations.ResolutionScope;
import org.apache.maven.project.MavenProject;
import org.yaml.snakeyaml.DumperOptions;
import org.yaml.snakeyaml.Yaml;
import org.yaml.snakeyaml.introspector.Property;
import org.yaml.snakeyaml.nodes.CollectionNode;
import org.yaml.snakeyaml.nodes.Node;
import org.yaml.snakeyaml.nodes.NodeTuple;
import org.yaml.snakeyaml.nodes.Tag;
import org.yaml.snakeyaml.representer.Representer;

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

        final Map<String, CamelArtifact> artifacts = new TreeMap<>();
        final CamelCatalog catalog = new DefaultCamelCatalog();

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

            DumperOptions options = new DumperOptions();
            options.setIndent(2);
            options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK);

            Representer representer = new CamelRepresenter();
            representer.addClassTag(CamelArtifact.class, Tag.MAP);

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
            //   artifacts:
            //
            try (Writer writer = Files.newBufferedWriter(output, StandardCharsets.UTF_8)) {
                Map<String, Object> cr = new LinkedHashMap<>();
                cr.put("apiVersion", "camel.apache.org/v1alpha1");
                cr.put("kind", "CamelCatalog");

                Map<String, Object> labels = new LinkedHashMap<>();
                labels.put("app", "camel-k");
                labels.put("camel.apache.org/catalog.version", catalog.getCatalogVersion());
                labels.put("camel.apache.org/catalog.loader.version", catalog.getLoadedVersion());

                Map<String, Object> meta = new LinkedHashMap<>();
                meta.put("name", "camel-catalog-" + catalog.getCatalogVersion());
                meta.put("labels", labels);

                Map<String, Object> spec = new LinkedHashMap<>();
                spec.put("artifacts", artifacts);
                spec.put("version", catalog.getCatalogVersion());

                cr.put("metadata", meta);
                cr.put("spec", spec);

                new Yaml(representer, options).dump(cr, writer);
            }
        } catch (IOException e) {
            throw new MojoExecutionException("Exception while generating catalog", e);
        }
    }

    private void processComponents(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
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

    private void processLanguages(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
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

    private void processDataFormats(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {
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

    // *************************
    //
    // Helpers
    //
    // *************************

    private static class CamelRepresenter extends Representer {
        @Override
        protected NodeTuple representJavaBeanProperty(Object javaBean, Property property, Object propertyValue, Tag customTag) {
            NodeTuple tuple = super.representJavaBeanProperty(javaBean, property, propertyValue, customTag);

            Node valueNode = tuple.getValueNode();
            if (Tag.NULL.equals(valueNode.getTag())) {
                return null;
            }
            if (valueNode instanceof CollectionNode) {
                CollectionNode col = (CollectionNode) valueNode;
                if (col.getValue().isEmpty()) {
                    return null;
                }
            }
            return tuple;
        }
    }
}

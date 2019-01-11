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
package org.apache.camel.k.tooling.maven.dependency;

import java.io.IOException;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.TreeMap;
import java.util.stream.Stream;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.camel.catalog.CamelCatalog;
import org.apache.camel.catalog.DefaultCamelCatalog;
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
import org.yaml.snakeyaml.nodes.Tag;
import org.yaml.snakeyaml.representer.Representer;

@Mojo(
    name = "generate-catalog",
    defaultPhase = LifecyclePhase.GENERATE_RESOURCES,
    threadSafe = true,
    requiresDependencyResolution = ResolutionScope.COMPILE_PLUS_RUNTIME,
    requiresDependencyCollection = ResolutionScope.COMPILE_PLUS_RUNTIME)
public class GenerateCatalogMojo extends AbstractMojo {
    private static final List<String> KNOWN_HTTP_URIS = Arrays.asList(
        "ahc",
        "ahc-ws",
        "atmosphere-websocket",
        "cxf",
        "cxfrs",
        "grpc",
        "jetty",
        "netty-http",
        "netty4-http",
        "rest",
        "restlet",
        "servlet",
        "spark-rest",
        "spring-ws",
        "undertow",
        "websocket",
        "knative"
    );

    private static final List<String> KNOWN_PASSIVE_URIS = Arrays.asList(
        "bean",
        "binding",
        "browse",
        "class",
        "controlbus",
        "dataformat",
        "dataset",
        "direct",
        "direct-vm",
        "language",
        "log",
        "mock",
        "properties",
        "ref",
        "seda",
        "stub",
        "test",
        "validator",
        "vm"
    );

    @Parameter(readonly = true, defaultValue = "${project}")
    private MavenProject project;

    @Parameter(property = "catalog.path", defaultValue = "${project.build.directory}/camel-catalog.yaml")
    private String outputFile;

    // ********************
    //
    // ********************

    @Override
    public void execute() throws MojoExecutionException, MojoFailureException {
        final Path output = Paths.get(this.outputFile);

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
            processEmbeddedComponent(catalog, artifacts);

            DumperOptions options = new DumperOptions();
            options.setIndent(2);
            options.setDefaultFlowStyle(DumperOptions.FlowStyle.BLOCK);

            Representer representer = new Representer();
            representer.addClassTag(GenerateCatalogMojo.CamelArtifact.class, Tag.MAP);

            try (Writer writer = Files.newBufferedWriter(output, StandardCharsets.UTF_8)) {
                Map<String, Object> answer = new HashMap<>();
                answer.put("artifacts", artifacts);
                answer.put("version", catalog.getLoadedVersion());

                new Yaml(representer, options).dump(answer, writer);
            }
        } catch (IOException e) {
            throw new MojoExecutionException("Exception while generating catalog", e);
        }
    }

    private void processComponents(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) throws IOException {
        ObjectMapper mapper = new ObjectMapper();

        for (String name : catalog.findComponentNames()) {
            String json = catalog.componentJSonSchema(name);
            ComponentDefinition definition = mapper.readValue(json, ComponentDefinitionContainer.class).getComponent();

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());
                    artifact.setVersion(definition.getVersion());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                    Objects.requireNonNull(artifact.getVersion());
                }

                definition.getSchemes()
                    .map(StringUtils::trimToNull)
                    .filter(Objects::nonNull)
                    .forEach(artifact::createScheme);

                return artifact;
            });
        }
    }

    private void processLanguages(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) throws IOException {
        ObjectMapper mapper = new ObjectMapper();

        for (String name : catalog.findLanguageNames()) {
            String json = catalog.languageJSonSchema(name);
            LanguageDefinition definition = mapper.readValue(json, LanguageDefinitionContainer.class).getLanguage();

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());
                    artifact.setVersion(definition.getVersion());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                    Objects.requireNonNull(artifact.getVersion());
                }

                artifact.addLanguage(definition.getName());

                return artifact;
            });
        }
    }

    private void processDataFormats(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) throws IOException {
        ObjectMapper mapper = new ObjectMapper();

        for (String name : catalog.findDataFormatNames()) {
            String json = catalog.dataFormatJSonSchema(name);
            DataFormatDefinition definition = mapper.readValue(json, DataFormatDefinitionContainer.class).getDataformat();

            artifacts.compute(definition.getArtifactId(), (key, artifact) -> {
                if (artifact == null) {
                    artifact = new CamelArtifact();
                    artifact.setGroupId(definition.getGroupId());
                    artifact.setArtifactId(definition.getArtifactId());
                    artifact.setVersion(definition.getVersion());

                    Objects.requireNonNull(artifact.getGroupId());
                    Objects.requireNonNull(artifact.getArtifactId());
                    Objects.requireNonNull(artifact.getVersion());
                }

                artifact.addDataformats(definition.getName());

                return artifact;
            });
        }
    }

    private void processEmbeddedComponent(CamelCatalog catalog, Map<String, CamelArtifact> artifacts) throws IOException {
        CamelArtifact knative = new CamelArtifact();
        knative.setGroupId("org.apache.camel.k");
        knative.setArtifactId("camel-knative");
        knative.setVersion(project.getVersion());
        knative.createScheme("knative").setHttp(true);

        artifacts.put(knative.getArtifactId(), knative);
    }

    // ********************
    // Model
    // ********************

    public static class CamelArtifact {
        private String groupId;
        private String artifactId;
        private String version;
        private List<CamelScheme> schemes;
        private List<String> languages;
        private List<String> dataformats;

        public CamelArtifact() {
            this.schemes = new ArrayList<>();
            this.languages = new ArrayList<>();
            this.dataformats = new ArrayList<>();
        }

        public String getGroupId() {
            return groupId;
        }

        public void setGroupId(String groupId) {
            this.groupId = groupId;
        }

        public String getArtifactId() {
            return artifactId;
        }

        public void setArtifactId(String artifactId) {
            this.artifactId = artifactId;
        }

        public String getVersion() {
            return version;
        }

        public void setVersion(String version) {
            this.version = version;
        }

        public void setSchemes(List<CamelScheme> schemes) {
            this.schemes = schemes;
        }

        public void addScheme(CamelScheme scheme) {
            if (!this.schemes.contains(scheme)) {
                this.schemes.add(scheme);
            }
        }

        public List<String> getLanguages() {
            return languages;
        }

        public void setLanguages(List<String> languages) {
            this.languages = languages;
        }

        public void addLanguage(String language) {
            if (!this.languages.contains(language)) {
                this.languages.add(language);
            }
        }

        public List<String> getDataformats() {
            return dataformats;
        }

        public void setDataformats(List<String> dataformats) {
            this.dataformats = dataformats;
        }

        public void addDataformats(String dataformat) {
            if (!this.dataformats.contains(dataformat)) {
                this.dataformats.add(dataformat);
            }
        }

        public List<CamelScheme> getSchemes() {
            return schemes;
        }

        public CamelScheme createScheme(String id) {
            for (CamelScheme scheme: schemes) {
                if (scheme.getId().equals(id)) {
                    return scheme;
                }
            }


            CamelScheme answer = new CamelScheme();
            answer.setId(id);
            answer.setHttp( KNOWN_HTTP_URIS.contains(id));
            answer.setPassive(KNOWN_PASSIVE_URIS.contains(id));

            schemes.add(answer);

            return answer;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) {
                return true;
            }
            if (o == null || getClass() != o.getClass()) {
                return false;
            }
            CamelArtifact artifact = (CamelArtifact) o;
            return Objects.equals(getArtifactId(), artifact.getArtifactId());
        }

        @Override
        public int hashCode() {
            return Objects.hash(getArtifactId());
        }
    }

    private static class CamelScheme {
        private String id;
        private boolean http;
        private boolean passive;

        public CamelScheme() {
        }

        public String getId() {
            return id;
        }

        public void setId(String id) {
            this.id = id;
        }

        public boolean isHttp() {
            return http;
        }

        public void setHttp(boolean http) {
            this.http = http;
        }

        public boolean isPassive() {
            return passive;
        }

        public void setPassive(boolean passive) {
            this.passive = passive;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) {
                return true;
            }
            if (o == null || getClass() != o.getClass()) {
                return false;
            }
            CamelScheme scheme = (CamelScheme) o;
            return isHttp() == scheme.isHttp() &&
                isPassive() == scheme.isPassive() &&
                Objects.equals(getId(), scheme.getId());
        }

        @Override
        public int hashCode() {
            return Objects.hash(getId(), isHttp(), isPassive());
        }
    }

    // ********************
    // Camel Catalog Model
    // ********************

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class ComponentDefinitionContainer {
        private ComponentDefinition component;

        public ComponentDefinition getComponent() {
            return component;
        }

        public void setComponent(ComponentDefinition component) {
            this.component = component;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class ComponentDefinition {
        private String scheme;
        private String groupId;
        private String artifactId;
        private String version;
        private String alternativeSchemes;

        public Stream<String> getSchemes() {
            String schemeIDs = StringUtils.trimToEmpty(alternativeSchemes);

            return Stream.concat(
                Stream.of(scheme),
                Stream.of(StringUtils.split(schemeIDs, ','))
            );
        }

        public String getScheme() {
            return scheme;
        }

        public void setScheme(String scheme) {
            this.scheme = scheme;
        }

        public String getGroupId() {
            return groupId;
        }

        public void setGroupId(String groupId) {
            this.groupId = groupId;
        }

        public String getArtifactId() {
            return artifactId;
        }

        public void setArtifactId(String artifactId) {
            this.artifactId = artifactId;
        }

        public String getVersion() {
            return version;
        }

        public void setVersion(String version) {
            this.version = version;
        }

        public String getAlternativeSchemes() {
            return alternativeSchemes;
        }

        public void setAlternativeSchemes(String alternativeSchemes) {
            this.alternativeSchemes = alternativeSchemes;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class LanguageDefinitionContainer {
        private LanguageDefinition language;

        public LanguageDefinition getLanguage() {
            return language;
        }

        public void setLanguage(LanguageDefinition language) {
            this.language = language;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class LanguageDefinition {
        private String name;
        private String groupId;
        private String artifactId;
        private String version;

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }

        public String getGroupId() {
            return groupId;
        }

        public void setGroupId(String groupId) {
            this.groupId = groupId;
        }

        public String getArtifactId() {
            return artifactId;
        }

        public void setArtifactId(String artifactId) {
            this.artifactId = artifactId;
        }

        public String getVersion() {
            return version;
        }

        public void setVersion(String version) {
            this.version = version;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class DataFormatDefinitionContainer {
        private DataFormatDefinition dataformat;

        public DataFormatDefinition getDataformat() {
            return dataformat;
        }

        public void setDataformat(DataFormatDefinition dataformat) {
            this.dataformat = dataformat;
        }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class DataFormatDefinition {
        private String name;
        private String groupId;
        private String artifactId;
        private String version;

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }

        public String getGroupId() {
            return groupId;
        }

        public void setGroupId(String groupId) {
            this.groupId = groupId;
        }

        public String getArtifactId() {
            return artifactId;
        }

        public void setArtifactId(String artifactId) {
            this.artifactId = artifactId;
        }

        public String getVersion() {
            return version;
        }

        public void setVersion(String version) {
            this.version = version;
        }
    }
}

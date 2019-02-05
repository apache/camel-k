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
package org.apache.camel.k.tooling.maven.model;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;
import java.util.Optional;

public class CamelArtifact extends Artifact {
    private List<CamelScheme> schemes;
    private List<String> languages;
    private List<String> dataformats;
    private List<Artifact> dependencies;
    private List<Artifact> exclusions;

    public CamelArtifact() {
        this.schemes = new ArrayList<>();
        this.languages = new ArrayList<>();
        this.dataformats = new ArrayList<>();
        this.dependencies = new ArrayList<>();
        this.exclusions = new ArrayList<>();
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

    public Optional<CamelScheme> getScheme(String id) {
        return schemes.stream().filter(s -> Objects.equals(s.getId(), id)).findFirst();
    }

    public CamelScheme createScheme(String id) {
        for (CamelScheme scheme: schemes) {
            if (scheme.getId().equals(id)) {
                return scheme;
            }
        }


        CamelScheme answer = new CamelScheme();
        answer.setId(id);

        schemes.add(answer);

        return answer;
    }

    public List<Artifact> getDependencies() {
        return dependencies;
    }

    public void setDependencies(List<Artifact> dependencies) {
        this.dependencies = dependencies;
    }

    public void addDependency(Artifact dependency) {
        if (!this.dependencies.contains(dependency)) {
            this.dependencies.add(dependency);
        }
    }

    public void addDependency(String groupId, String artifactId) {
        Artifact artifact = new Artifact();
        artifact.setGroupId(groupId);
        artifact.setArtifactId(artifactId);

        addDependency(artifact);
    }

    public void addDependency(String groupId, String artifactId, String version) {
        Artifact artifact = new Artifact();
        artifact.setGroupId(groupId);
        artifact.setArtifactId(artifactId);
        artifact.setVersion(version);

        addDependency(artifact);
    }

    public List<Artifact> getExclusions() {
        return exclusions;
    }

    public void setExclusions(List<Artifact> exclusions) {
        this.exclusions = exclusions;
    }

    public void addExclusion(Artifact exclusion) {
        if (!this.exclusions.contains(exclusion)) {
            this.exclusions.add(exclusion);
        }
    }

    public void addExclusion(String groupId, String artifactId) {
        Artifact artifact = new Artifact();
        artifact.setGroupId(groupId);
        artifact.setArtifactId(artifactId);

        addExclusion(artifact);
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

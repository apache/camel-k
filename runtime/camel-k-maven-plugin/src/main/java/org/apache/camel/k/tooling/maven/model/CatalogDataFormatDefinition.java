package org.apache.camel.k.tooling.maven.model;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonIgnoreProperties(ignoreUnknown = true)
public final class CatalogDataFormatDefinition {
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

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class Container {
        private CatalogDataFormatDefinition delegate;

        @JsonCreator
        public Container(
            @JsonProperty("dataformat") CatalogDataFormatDefinition delegate) {
            this.delegate = delegate;
        }

        public CatalogDataFormatDefinition unwrap() {
            return delegate;
        }
    }
}

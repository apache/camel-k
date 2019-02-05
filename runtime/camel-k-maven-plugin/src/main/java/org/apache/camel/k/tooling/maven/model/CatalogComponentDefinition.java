package org.apache.camel.k.tooling.maven.model;

import java.util.stream.Stream;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import org.apache.commons.lang3.StringUtils;

@JsonIgnoreProperties(ignoreUnknown = true)
public final class CatalogComponentDefinition {
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

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static final class Container {
        private CatalogComponentDefinition delegate;

        @JsonCreator
        public Container(
            @JsonProperty("component") CatalogComponentDefinition delegate) {
            this.delegate = delegate;
        }

        public CatalogComponentDefinition unwrap() {
            return delegate;
        }
    }
}

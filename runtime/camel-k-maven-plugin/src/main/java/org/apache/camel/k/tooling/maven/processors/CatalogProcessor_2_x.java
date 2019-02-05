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
package org.apache.camel.k.tooling.maven.processors;

import java.util.Arrays;
import java.util.List;
import java.util.Map;

import com.vdurmont.semver4j.Semver;

import org.apache.camel.Ordered;
import org.apache.camel.catalog.CamelCatalog;
import org.apache.camel.k.tooling.maven.model.CamelArtifact;
import org.apache.camel.k.tooling.maven.model.CatalogProcessor;
import org.apache.maven.project.MavenProject;

public class CatalogProcessor_2_x implements CatalogProcessor {
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
        "websocket"
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

    @Override
    public int getOrder() {
        return HIGHEST;
    }

    @Override
    public boolean accepts(CamelCatalog catalog) {
        return new Semver(catalog.getCatalogVersion(), Semver.SemverType.IVY).satisfies("[2.18,3]");
    }

    @Override
    public void process(MavenProject project, CamelCatalog catalog, Map<String, CamelArtifact> artifacts) {

        // ************************
        //
        // camel-k-runtime-jvm
        //
        // ************************

        {
            CamelArtifact artifact = new CamelArtifact();
            artifact.setGroupId("org.apache.camel.k");
            artifact.setArtifactId("camel-k-runtime-jvm");
            artifact.setVersion(project.getVersion());
            artifact.addDependency("org.apache.camel", "camel-core");

            artifacts.put(artifact.getArtifactId(), artifact);
        }

        // ************************
        //
        // camel-k-runtime-groovy
        //
        // ************************

        {
            CamelArtifact artifact = new CamelArtifact();
            artifact.setGroupId("org.apache.camel.k");
            artifact.setArtifactId("camel-k-runtime-groovy");
            artifact.setVersion(project.getVersion());
            artifact.addDependency("org.apache.camel", "camel-groovy");

            artifacts.put(artifact.getArtifactId(), artifact);
        }

        // ************************
        //
        // camel-k-runtime-kotlin
        //
        // ************************

        {
            CamelArtifact artifact = new CamelArtifact();
            artifact.setGroupId("org.apache.camel.k");
            artifact.setArtifactId("camel-k-runtime-kotlin");
            artifact.setVersion(project.getVersion());

            artifacts.put(artifact.getArtifactId(), artifact);
        }

        // ************************
        //
        // camel-knative
        //
        // ************************

        {
            CamelArtifact artifact = new CamelArtifact();
            artifact.setGroupId("org.apache.camel.k");
            artifact.setArtifactId("camel-knative");
            artifact.setVersion(project.getVersion());
            artifact.createScheme("knative").setHttp(true);
            artifact.addDependency("org.apache.camel", "camel-netty4-http");

            artifacts.put(artifact.getArtifactId(), artifact);
        }

        // ************************
        //
        //
        //
        // ************************

        for (String scheme: KNOWN_HTTP_URIS) {
            artifacts.values().forEach(artifact -> artifact.getScheme(scheme).ifPresent(s -> s.setHttp(true)));
        }
        for (String scheme: KNOWN_PASSIVE_URIS) {
            artifacts.values().forEach(artifact -> artifact.getScheme(scheme).ifPresent(s -> s.setPassive(true)));
        }
    }
}

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

import java.util.Map;

import org.apache.camel.catalog.CamelCatalog;
import org.apache.maven.project.MavenProject;

public interface CatalogProcessor {
    /**
     * The highest precedence
     */
    int HIGHEST = Integer.MIN_VALUE;

    /**
     * The lowest precedence
     */
    int LOWEST = Integer.MAX_VALUE;

    boolean accepts(CamelCatalog catalog);

    void process(MavenProject project, CamelCatalog catalog, Map<String, CamelArtifact> artifacts);

    default int getOrder() {
        return LOWEST;
    }
}

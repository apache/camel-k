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
package org.apache.camel.k;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;

import org.apache.camel.util.ObjectHelper;
import org.apache.commons.lang3.StringUtils;

public enum Language {
    Unknown(
        "unknown",
        Collections.emptyList(),
        Collections.emptyList()),
    JavaClass(
        "java-class",
        Collections.singletonList("class"),
        Collections.singletonList("class")),
    JavaSource(
        "java-source",
        Collections.singletonList("java"),
        Collections.singletonList("java")),
    JavaScript(
        "js",
        Arrays.asList("js", "javascript"),
        Collections.singletonList("js")),
    Groovy(
        "groovy",
        Collections.singletonList("groovy"),
        Collections.singletonList("groovy")),
    Xml(
        "xml",
        Collections.singletonList("xml"),
        Collections.singletonList("xml")),
    Kotlin(
        "kotlin",
        Arrays.asList("kotlin", "kts"),
        Collections.singletonList("kts"));

    private final String id;
    private final List<String> names;
    private final List<String> extensions;

    Language(String id, List<String> names, List<String> extensions) {
        this.id = ObjectHelper.notNull(id, "id");
        this.names = names;
        this.extensions = extensions;
    }

    public String getId() {
        return id;
    }

    public List<String> getNames() {
        return names;
    }

    public List<String> getExtensions() {
        return extensions;
    }

    public static Language fromLanguageName(String name) {
        for (Language language: values()) {
            if (language.getNames().contains(name)) {
                return language;
            }
        }

        return Unknown;
    }

    public static Language fromResource(String resource) {
        for (Language language: values()) {
            String path = StringUtils.substringAfter(resource, ":");

            for (String ext : language.getExtensions()) {
                if (path.endsWith("." + ext)) {
                    return language;
                }
            }
        }

        return Unknown;
    }
}

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
package org.apache.camel.k.jvm;

import java.util.Collections;
import java.util.List;

import org.apache.commons.lang3.StringUtils;

public enum Language {
    Unknow("", Collections.emptyList()),
    JavaClass("class", Collections.singletonList("class")),
    JavaSource("java", Collections.singletonList("java")),
    JavaScript("js", Collections.singletonList("js")),
    Groovy("groovy", Collections.singletonList("groovy")),
    Xml("xml", Collections.singletonList("xml"));

    private final String name;
    private final List<String> extensions;

    Language(String name, List<String> extensions) {
        this.name = name;
        this.extensions = extensions;
    }

    public String getName() {
        return name;
    }

    public List<String> getExtensions() {
        return extensions;
    }

    public static Language fromLanguageName(String name) {
        for (Language language: values()) {
            if (language.getName().equals(name)) {
                return language;
            }
        }

        return Unknow;
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

        return Unknow;
    }
}

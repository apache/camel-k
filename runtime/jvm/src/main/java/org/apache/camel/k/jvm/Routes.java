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

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Paths;

import org.apache.camel.util.ObjectHelper;
import org.apache.commons.lang3.StringUtils;

public final class Routes {
    public static final String ENV_CAMEL_K_ROUTES_URI = "CAMEL_K_ROUTES_URI";
    public static final String ENV_CAMEL_K_ROUTES_LANGUAGE = "CAMEL_K_ROUTES_LANGUAGE";
    public static final String ENV_CAMEL_K_CONF = "CAMEL_K_CONF";
    public static final String ENV_CAMEL_K_CONF_D = "CAMEL_K_CONF_D";
    public static final String SCHEME_CLASSPATH = "classpath:";
    public static final String SCHEME_FILE = "file:";

    private Routes() {
    }

    public static boolean isScripting(String resource) {
        return resource.endsWith(".java") || resource.endsWith(".js") || resource.endsWith(".groovy") || resource.endsWith(".xml");
    }

    public static RoutesLoader loaderForLanguage(String language) {
        for (RoutesLoader loader: RoutesLoaders.values()) {
            if (loader.getSupportedLanguages().contains(language)) {
                return loader;
            }
        }

        throw new IllegalArgumentException("Unable to find loader for language: " + language);
    }

    public static RoutesLoader loaderForResource(String resource) {
        if (!resource.startsWith(SCHEME_CLASSPATH) && !resource.startsWith(SCHEME_FILE)) {
            throw new IllegalArgumentException("No valid resource format, expected scheme:path, found " + resource);
        }

        for (RoutesLoader loader: RoutesLoaders.values()) {
            if (loader.test(resource)) {
                return loader;
            }
        }

        throw new IllegalArgumentException("Unable to find loader for: " + resource);
    }

    public static RoutesLoader loaderFor(String resource, String language) {
        if (!resource.startsWith(SCHEME_CLASSPATH) && !resource.startsWith(SCHEME_FILE)) {
            throw new IllegalArgumentException("No valid resource format, expected scheme:path, found " + resource);
        }

        return ObjectHelper.isEmpty(language)
            ? loaderForResource(resource)
            : loaderForLanguage(language);
    }

    static InputStream loadResourceAsInputStream(String resource) throws IOException {
        if (resource.startsWith(SCHEME_CLASSPATH)) {
            String location = StringUtils.removeStart(resource, SCHEME_CLASSPATH);
            if (!location.startsWith("/")) {
                location = "/" + location;
            }

            return Application.class.getResourceAsStream(location);
        } else {
            return Files.newInputStream(
                Paths.get(resource.substring(SCHEME_FILE.length()))
            );
        }
    }
}

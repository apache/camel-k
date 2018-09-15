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
import java.nio.file.FileVisitResult;
import java.nio.file.FileVisitor;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.List;
import java.util.Objects;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.main.Main;
import org.apache.camel.util.ObjectHelper;
import org.apache.commons.io.FilenameUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Application {
    private static final Logger LOGGER = LoggerFactory.getLogger(Application.class);

    public static void main(String[] args) throws Exception {
        final String resource = System.getenv(Constants.ENV_CAMEL_K_ROUTES_URI);
        final String language = System.getenv(Constants.ENV_CAMEL_K_ROUTES_LANGUAGE);

        if (ObjectHelper.isEmpty(resource)) {
            throw new IllegalStateException("No valid resource found in " + Constants.ENV_CAMEL_K_ROUTES_URI + " environment variable");
        }

        String locations = computePropertyPlaceholderLocations();
        RoutesLoader loader = RoutesLoaders.loaderFor(resource, language);
        RouteBuilder routes = loader.load(resource);

        if (routes == null) {
            throw new IllegalStateException("Unable to load route from: " + resource);
        }

        LOGGER.info("Routes: {}", resource);
        LOGGER.info("Language: {}", language);
        LOGGER.info("Locations: {}", locations);

        Main main = new Main();

        if (ObjectHelper.isNotEmpty(locations)) {
            main.setPropertyPlaceholderLocations(locations);
        }

        main.addRouteBuilder(routes);
        main.run();
    }

    // *******************************
    //
    // helpers
    //
    // *******************************

    private static String computePropertyPlaceholderLocations() throws IOException {
        final String conf = System.getenv(Constants.ENV_CAMEL_K_CONF);
        final String confd = System.getenv(Constants.ENV_CAMEL_K_CONF_D);
        final List<String> locations = new ArrayList<>();

        // Main location
        if (ObjectHelper.isNotEmpty(conf)) {
            locations.add("file:" + conf);
        }

        // Additional locations
        if (ObjectHelper.isNotEmpty(confd)) {
            Path root = Paths.get(confd);
            FileVisitor<Path> visitor = new SimpleFileVisitor<Path>() {
                @Override
                public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
                    Objects.requireNonNull(file);
                    Objects.requireNonNull(attrs);

                    String path = file.toFile().getAbsolutePath();
                    String ext = FilenameUtils.getExtension(path);

                    if (Objects.equals("properties", ext)) {
                        locations.add("file:" + path);
                    }

                    return FileVisitResult.CONTINUE;
                }
            };

            if (Files.exists(root)) {
                Files.walkFileTree(root, visitor);
            }
        }

        return String.join(",", locations);
    }
}

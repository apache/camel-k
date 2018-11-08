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
import java.io.Reader;
import java.nio.file.FileVisitResult;
import java.nio.file.FileVisitor;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.Objects;
import java.util.Properties;

import org.apache.camel.util.IntrospectionSupport;
import org.apache.camel.util.ObjectHelper;
import org.apache.commons.io.FilenameUtils;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.LoggerConfig;

import static org.apache.camel.k.jvm.Constants.SCHEME_INLINE;

public final class RuntimeSupport {
    private RuntimeSupport() {
    }

    public static void configureSystemProperties() {
        final String conf = System.getenv(Constants.ENV_CAMEL_K_CONF);
        final String confd = System.getenv(Constants.ENV_CAMEL_K_CONF_D);
        final Properties properties = new Properties();

        // Main location
        if (ObjectHelper.isNotEmpty(conf)) {
            if (conf.startsWith(SCHEME_INLINE)) {
                try (Reader reader = URIResolver.resolveInline(conf)) {
                    properties.load(reader);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            } else {
                try (Reader reader = Files.newBufferedReader(Paths.get(conf))) {
                    properties.load(reader);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }
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
                        try (Reader reader = Files.newBufferedReader(Paths.get(path))) {
                            properties.load(reader);
                        }
                    }

                    return FileVisitResult.CONTINUE;
                }
            };

            if (Files.exists(root)) {
                try {
                    Files.walkFileTree(root, visitor);
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }
        }

        System.getProperties().putAll(properties);
    }

    public static void configureLogging() {
        final LoggerContext ctx = (LoggerContext) LogManager.getContext(false);
        final Properties properties = System.getProperties();

        properties.entrySet().stream()
            .filter(entry -> entry.getKey() instanceof String)
            .filter(entry -> entry.getValue() instanceof String)
            .filter(entry -> ((String)entry.getKey()).startsWith(Constants.LOGGING_LEVEL_PREFIX))
            .forEach(entry -> {
                String key = ((String)entry.getKey());
                String val = ((String)entry.getValue());

                String logger = key.substring(Constants.LOGGING_LEVEL_PREFIX.length());
                Level level = Level.getLevel(val);
                LoggerConfig config = new LoggerConfig(logger, level, true);

                ctx.getConfiguration().addLogger(logger, config);
            }
        );
    }

    public static void bindProperties(Object target, String prefix) {
        // Integration properties are defined as system properties
        final Properties properties = System.getProperties();

        properties.entrySet().stream()
            .filter(entry -> entry.getKey() instanceof String)
            .filter(entry -> entry.getValue() != null)
            .filter(entry -> ((String)entry.getKey()).startsWith(prefix))
            .forEach(entry -> {
                    final String key = ((String)entry.getKey()).substring(prefix.length());
                    final Object val = entry.getValue();

                    try {
                        IntrospectionSupport.setProperty(target, key, val, false);
                    } catch (Exception ex) {
                        throw new RuntimeException(ex);
                    }
                }
            );
    }
}

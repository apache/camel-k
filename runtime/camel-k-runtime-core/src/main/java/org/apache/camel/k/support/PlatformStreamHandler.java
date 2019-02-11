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
package org.apache.camel.k.support;

import java.io.ByteArrayInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.net.URISyntaxException;
import java.net.URL;
import java.net.URLConnection;
import java.net.URLStreamHandler;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Base64;
import java.util.Map;
import java.util.zip.GZIPInputStream;

import org.apache.camel.util.ObjectHelper;
import org.apache.camel.util.URISupport;
import org.apache.commons.lang3.StringUtils;

public class PlatformStreamHandler extends URLStreamHandler {
    public static void configure() {
        URL.setURLStreamHandlerFactory(protocol -> "platform".equals(protocol) ? new PlatformStreamHandler() : null);
    }

    @Override
    protected URLConnection openConnection(URL url) throws IOException {
        return new URLConnection(url) {
            @Override
            public void connect() throws IOException {
            }

            @Override
            public InputStream getInputStream() throws IOException {
                InputStream is = null;

                // check if the file exists
                Path path = Paths.get(url.getPath());
                if (Files.exists(path)) {
                    is = Files.newInputStream(path);
                }

                // check if the file exists in classpath
                if (is == null) {
                    is = ObjectHelper.loadResourceAsStream(url.getPath());
                }

                if (is == null) {
                    String name = getURL().getPath().toUpperCase();
                    name = name.replace(" ", "_");
                    name = name.replace(".", "_");
                    name = name.replace("-", "_");

                    String envName = System.getenv(name);
                    String envType = StringUtils.substringBefore(envName, ":");
                    String envQuery = StringUtils.substringAfter(envName, "?");

                    envName = StringUtils.substringAfter(envName, ":");
                    envName = StringUtils.substringBefore(envName, "?");

                    if (envName != null) {
                        try {
                            final Map<String, Object> params = URISupport.parseQuery(envQuery);
                            final boolean compression = Boolean.valueOf((String) params.get("compression"));

                            if (StringUtils.equals(envType, "env")) {
                                String data = System.getenv(envName);

                                if (data == null) {
                                    throw new IllegalArgumentException("Unknown env var: " + envName);
                                }

                                is = new ByteArrayInputStream(data.getBytes());
                            } else if (StringUtils.equals(envType, "file")) {
                                Path data = Paths.get(envName);

                                if (!Files.exists(data)) {
                                    throw new FileNotFoundException(envName);
                                }

                                is = Files.newInputStream(data);
                            } else if (StringUtils.equals(envType, "classpath")) {
                                is = ObjectHelper.loadResourceAsStream(envName);
                            }

                            if (is != null && compression) {
                                is = new GZIPInputStream(Base64.getDecoder().wrap(is));
                            }
                        } catch (URISyntaxException e) {
                            throw new IOException(e);
                        }
                    }
                }

                return is;
            }
        };
    }
}

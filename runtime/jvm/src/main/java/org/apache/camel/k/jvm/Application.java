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

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.main.Main;

public class Application {
    public static final String ENV_CAMEL_K_ROUTES_URI = "CAMEL_K_ROUTES_URI";
    public static final String SCHEME_CLASSPATH = "classpath:";
    public static final String SCHEME_FILE = "file:";

    public static void main(String[] args) throws Exception {
        final String resource = System.getenv(ENV_CAMEL_K_ROUTES_URI);

        if (resource == null || resource.trim().length() == 0) {
            throw new IllegalStateException("No valid resource found in " + ENV_CAMEL_K_ROUTES_URI + " environment variable");
        }
        if (!resource.startsWith(SCHEME_CLASSPATH) && !resource.startsWith(SCHEME_FILE)) {
            throw new IllegalStateException("No valid resource format, expected scheme:path, found " + resource);
        }

        RoutesLoader loader = RouteLoaders.loaderFor(resource);
        RouteBuilder routes = loader.load(resource);

        if (routes == null) {
            throw new IllegalStateException("Unable to load route from: " + resource);
        }

        Main main = new Main();
        main.addRouteBuilder(routes);
        main.run();
    }
}

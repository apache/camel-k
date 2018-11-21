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

public final class Constants {
    public static final String ENV_CAMEL_K_ROUTES = "CAMEL_K_ROUTES";
    public static final String ENV_CAMEL_K_CONF = "CAMEL_K_CONF";
    public static final String ENV_CAMEL_K_CONF_D = "CAMEL_K_CONF_D";
    public static final String SCHEME_CLASSPATH = "classpath:";
    public static final String SCHEME_FILE = "file:";
    public static final String SCHEME_ENV = "env:";
    public static final String LOGGING_LEVEL_PREFIX = "logging.level.";

    private Constants() {
    }
}

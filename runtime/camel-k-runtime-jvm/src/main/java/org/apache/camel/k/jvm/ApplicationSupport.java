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

import java.util.Properties;

import org.apache.camel.k.Constants;
import org.apache.camel.k.support.RuntimeSupport;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.LoggerConfig;


public final class ApplicationSupport {
    private ApplicationSupport() {
    }

    public static void configureLogging() {
        final LoggerContext ctx = (LoggerContext) LogManager.getContext(false);
        final Properties properties = RuntimeSupport.loadProperties();

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
}

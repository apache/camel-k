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

import java.util.List;
import java.util.function.Predicate;

import org.apache.camel.builder.RouteBuilder;

public interface RoutesLoader extends Predicate<String> {
    /**
     * Provides a list of the languages supported by this loader.
     *
     * @return the supported languages.
     */
    List<String> getSupportedLanguages();

    /**
     * Creates a camel {@link RouteBuilder} from the given resource.
     *
     * @param resource the location fo the resource to load.
     * @return the RouteBuilder.
     * @throws Exception
     */
    RouteBuilder load(String resource) throws Exception;
}

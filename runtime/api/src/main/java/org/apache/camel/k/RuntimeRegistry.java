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

import java.util.Map;

import org.apache.camel.spi.Registry;

public interface RuntimeRegistry extends Registry {
    void bind(String name, Object bean);

    @SuppressWarnings("deprecation")
    @Override
    default public Object lookup(String name) {
        return lookupByName(name);
    }

    @SuppressWarnings("deprecation")
    @Override
    default public <T> T lookup(String name, Class<T> type) {
        return lookupByNameAndType(name, type);
    }

    @SuppressWarnings("deprecation")
    @Override
    default public <T> Map<String, T> lookupByType(Class<T> type) {
        return findByTypeWithName(type);
    }
}

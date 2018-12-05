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

import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.stream.Collectors;

import org.apache.camel.NoSuchBeanException;

public class SimpleRuntimeRegistry implements RuntimeRegistry {
    private final ConcurrentMap<String, Object> registry;

    public SimpleRuntimeRegistry() {
        this.registry = new ConcurrentHashMap<>();
    }

    public void bind(String name, Object bean) {
        this.registry.put(name, bean);
    }

    @Override
    public Object lookupByName(String name) {
        return registry.get(name);
    }

    @Override
    public <T> T lookupByNameAndType(String name, Class<T> type) {
        final Object answer = lookupByName(name);

        if (answer != null) {
            try {
                return type.cast(answer);
            } catch (Throwable t) {
                throw new NoSuchBeanException(
                    name,
                    "Found bean: " + name + " in RuntimeRegistry: " + this + " of type: " + answer.getClass().getName() + " expected type was: " + type,
                    t
                );
            }
        }

        return null;
    }

    @Override
    public <T> Map<String, T> findByTypeWithName(Class<T> type) {
        final Map<String, T> result = new HashMap<>();

        registry.entrySet().stream()
            .filter(entry -> type.isInstance(entry.getValue()))
            .forEach(entry -> result.put(entry.getKey(), type.cast(entry.getValue())));

        return result;
    }

    @Override
    public <T> Set<T> findByType(Class<T> type) {
        return registry.values().stream()
            .filter(type::isInstance)
            .map(type::cast)
            .collect(Collectors.toSet());
    }
}

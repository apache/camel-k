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

import org.apache.camel.util.ObjectHelper;
import org.apache.camel.util.URISupport;
import org.apache.commons.lang3.StringUtils;

public class Source {
    private final String location;
    private final Language language;
    private final boolean compressed;

    private Source(String location, Language language, boolean compression) {
        this.location = location;
        this.language = language;
        this.compressed = compression;
    }

    public String getLocation() {
        return location;
    }

    public Language getLanguage() {
        return language;
    }

    public boolean isCompressed() {
        return compressed;
    }

    @Override
    public String toString() {
        return "Source{" +
            "location='" + location + '\'' +
            ", language=" + language +
            ", compressed=" + compressed +
            '}';
    }

    public static Source create(String uri) throws Exception {
        final String location = StringUtils.substringBefore(uri, "?");

        if (!location.startsWith(Constants.SCHEME_CLASSPATH) &&
            !location.startsWith(Constants.SCHEME_FILE) &&
            !location.startsWith(Constants.SCHEME_ENV)) {
            throw new IllegalArgumentException("No valid resource format, expected scheme:path, found " + uri);
        }

        final String query = StringUtils.substringAfter(uri, "?");
        final Map<String, Object> params = URISupport.parseQuery(query);
        final String languageName = (String) params.get("language");
        final boolean compression = Boolean.valueOf((String) params.get("compression"));

        Language language = ObjectHelper.isNotEmpty(languageName)
            ? Language.fromLanguageName(languageName)
            : Language.fromResource(location);

        return new Source(location, language, compression);
    }
}

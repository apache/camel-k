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
package org.apache.camel.component.knative;

import java.io.UnsupportedEncodingException;
import java.net.URISyntaxException;
import java.util.HashMap;
import java.util.Map;
import java.util.Objects;

import org.apache.camel.Exchange;
import org.apache.camel.util.CollectionHelper;
import org.apache.camel.util.URISupport;

public final class KnativeSupport {
    private KnativeSupport() {
    }

    public static boolean hasStructuredContent(Exchange exchange) {
        return Objects.equals(exchange.getIn().getHeader(Exchange.CONTENT_TYPE), Knative.MIME_STRUCTURED_CONTENT_MODE);
    }

    public static <K, V> Map<K, V> mergeMaps(Map<K, V> map, Map<K, V>... maps) {
        Map<K, V> answer = new HashMap<>();

        if (map != null) {
            answer.putAll(map);
        }

        for (Map<K, V> m : maps) {
            answer.putAll(m);
        }

        return answer;
    }

    public static String appendParametersToURI(String uri, String key, Object value, Object... keyVals)
            throws UnsupportedEncodingException, URISyntaxException {

        return URISupport.appendParametersToURI(
            uri,
            CollectionHelper.mapOf(key, value, keyVals)
        );
    }
}

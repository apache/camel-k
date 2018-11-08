/**
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 * <p>
 * http://www.apache.org/licenses/LICENSE-2.0
 * <p>
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.apache.camel.k.jvm;

import org.apache.camel.CamelContext;
import org.apache.camel.util.ResourceHelper;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.Reader;
import java.io.StringReader;

import static org.apache.camel.k.jvm.Constants.SCHEME_INLINE;

public class URIResolver {

    public static InputStream resolve(CamelContext ctx, String uri) throws IOException {
        if (uri == null) {
            throw new IllegalArgumentException("Cannot resolve null URI");
        }
        if (uri.startsWith(SCHEME_INLINE)) {
            // Using platform encoding on purpose
            return new ByteArrayInputStream(uri.substring(SCHEME_INLINE.length()).getBytes());
        }

        return ResourceHelper.resolveMandatoryResourceAsInputStream(ctx, uri);
    }

    public static Reader resolveInline(String uri) {
        if (!uri.startsWith(SCHEME_INLINE)) {
            throw new IllegalArgumentException("The provided content is not inline: " + uri);
        }
        return new StringReader(uri.substring(SCHEME_INLINE.length()));
    }

}

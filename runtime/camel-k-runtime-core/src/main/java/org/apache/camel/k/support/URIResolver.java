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
package org.apache.camel.k.support;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.io.Reader;
import java.io.StringReader;
import java.util.Base64;
import java.util.zip.GZIPInputStream;

import org.apache.camel.CamelContext;
import org.apache.camel.k.Constants;
import org.apache.camel.k.Source;
import org.apache.camel.util.ResourceHelper;
import org.apache.camel.util.StringHelper;


public class URIResolver {

    public static InputStream resolve(CamelContext ctx, Source source) throws Exception {
        if (source.getLocation() == null) {
            throw new IllegalArgumentException("Cannot resolve null URI");
        }

        final InputStream is;

        if (source.getLocation().startsWith(Constants.SCHEME_ENV)) {
            final String envvar = StringHelper.after(source.getLocation(), ":");
            final String content = System.getenv(envvar);

            // Using platform encoding on purpose
            is = new ByteArrayInputStream(content.getBytes());
        } else {
            is = ResourceHelper.resolveMandatoryResourceAsInputStream(ctx, source.getLocation());
        }

        return source.isCompressed() ? new GZIPInputStream(Base64.getDecoder().wrap(is)) : is;
    }

    public static Reader resolveEnv(String uri) {
        if (!uri.startsWith(Constants.SCHEME_ENV)) {
            throw new IllegalArgumentException("The provided content is not env: " + uri);
        }

        final String envvar = StringHelper.after(uri, ":");
        final String content = System.getenv(envvar);

        return new StringReader(content);
    }

}

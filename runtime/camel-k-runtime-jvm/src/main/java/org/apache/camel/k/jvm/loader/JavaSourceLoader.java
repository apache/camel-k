package org.apache.camel.k.jvm.loader;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.List;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.Language;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.Source;
import org.apache.camel.k.support.URIResolver;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.joor.Reflect;

public class JavaSourceLoader implements RoutesLoader {
    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.JavaSource);
    }

    @Override
    public RouteBuilder load(RuntimeRegistry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                try (InputStream is = URIResolver.resolve(getContext(), source)) {
                    String name = StringUtils.substringAfter(source.getLocation(), ":");
                    name = StringUtils.removeEnd(name, ".java");

                    if (name.contains("/")) {
                        name = StringUtils.substringAfterLast(name, "/");
                    }

                    // Wrap routes builder
                    includeRoutes(
                        Reflect.compile(name, IOUtils.toString(is, StandardCharsets.UTF_8)).create().get()
                    );
                }
            }
        };
    }
}

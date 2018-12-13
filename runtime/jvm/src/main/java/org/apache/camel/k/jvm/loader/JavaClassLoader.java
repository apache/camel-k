package org.apache.camel.k.jvm.loader;

import java.util.Collections;
import java.util.List;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.Constants;
import org.apache.camel.k.Language;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.Source;
import org.apache.commons.lang3.StringUtils;

public class JavaClassLoader implements RoutesLoader {
    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.JavaClass);
    }

    @Override
    public RouteBuilder load(RuntimeRegistry registry, Source source) throws Exception {
        String path = source.getLocation();
        path = StringUtils.removeStart(path, Constants.SCHEME_CLASSPATH);
        path = StringUtils.removeEnd(path, ".class");

        Class<?> type = Class.forName(path);

        if (!RouteBuilder.class.isAssignableFrom(type)) {
            throw new IllegalStateException("The class provided (" + path + ") is not a org.apache.camel.builder.RouteBuilder");
        }

        return (RouteBuilder)type.newInstance();
    }
}

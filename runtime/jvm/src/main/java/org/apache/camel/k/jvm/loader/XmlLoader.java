package org.apache.camel.k.jvm.loader;

import java.io.InputStream;
import java.util.Collections;
import java.util.List;
import javax.xml.bind.UnmarshalException;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.k.Language;
import org.apache.camel.k.RoutesLoader;
import org.apache.camel.k.RuntimeRegistry;
import org.apache.camel.k.Source;
import org.apache.camel.k.jvm.URIResolver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class XmlLoader implements RoutesLoader {
    private static final Logger LOGGER = LoggerFactory.getLogger(XmlLoader.class);

    @Override
    public List<Language> getSupportedLanguages() {
        return Collections.singletonList(Language.Xml);
    }

    @Override
    public RouteBuilder load(RuntimeRegistry registry, Source source) throws Exception {
        return new RouteBuilder() {
            @Override
            public void configure() throws Exception {
                try (InputStream is = URIResolver.resolve(getContext(), source)) {
                    try {
                        setRouteCollection(
                            getContext().loadRoutesDefinition(is)
                        );
                    } catch (UnmarshalException e) {
                        LOGGER.debug("Unable to load RoutesDefinition: {}", e.getMessage());
                    }

                    try {
                        setRestCollection(
                            getContext().loadRestsDefinition(is)
                        );
                    } catch (UnmarshalException e) {
                        LOGGER.debug("Unable to load RestsDefinition: {}", e.getMessage());
                    }
                }
            }
        };
    }
}

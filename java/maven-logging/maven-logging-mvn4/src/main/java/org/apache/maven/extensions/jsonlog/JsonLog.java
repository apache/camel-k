package org.apache.maven.extensions.jsonlog;

import org.apache.maven.cli.logging.Slf4jLoggerManager;
import org.apache.maven.eventspy.AbstractEventSpy;
import org.apache.maven.rtinfo.RuntimeInformation;
import org.codehaus.plexus.MutablePlexusContainer;
import org.codehaus.plexus.PlexusContainer;
import org.slf4j.LoggerFactory;

import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;
import java.lang.reflect.Field;

@Named
@Singleton
public class JsonLog extends AbstractEventSpy {

    @Inject
    JsonLog(RuntimeInformation runtimeInformation, PlexusContainer container) throws Exception {
        try {
            // Maven 4 extension code:
            if (runtimeInformation.getMavenVersion().startsWith("4.")) {
                org.slf4j.spi.SLF4JServiceProvider provider = new ch.qos.logback.classic.spi.LogbackServiceProvider();
                provider.initialize();
                Field field = LoggerFactory.class.getDeclaredField("PROVIDER");
                field.setAccessible(true);
                field.set(null, provider);

                if (container instanceof MutablePlexusContainer) {
                    ((MutablePlexusContainer) container).setLoggerManager(new Slf4jLoggerManager());
                }

                LoggerFactory.getLogger(getClass()).debug("Json Logging initialized");
            }
        } catch (Exception e) {
            e.printStackTrace();
            throw e;
        }
    }

}

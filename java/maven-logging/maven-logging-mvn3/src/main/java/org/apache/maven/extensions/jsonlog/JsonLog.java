package org.apache.maven.extensions.jsonlog;

import ch.qos.logback.classic.util.ContextInitializer;
import org.apache.maven.cli.logging.Slf4jLoggerManager;
import org.apache.maven.eventspy.AbstractEventSpy;
import org.apache.maven.rtinfo.RuntimeInformation;
import org.codehaus.plexus.MutablePlexusContainer;
import org.codehaus.plexus.PlexusContainer;
import org.codehaus.plexus.logging.LoggerManager;
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
            // Maven 3 extension code
            if (runtimeInformation.getMavenVersion().startsWith("3.")) {
                Class<?> binderClass = org.slf4j.LoggerFactory.class.getClassLoader().loadClass(
                    org.slf4j.impl.StaticLoggerBinder.class.getName());
                System.err.println(binderClass.getClassLoader().getResource(binderClass.getName().replace('.','/') + ".class"));
                Object binder = binderClass.getMethod("getSingleton").invoke(null);

                Field field = binder.getClass().getDeclaredField("loggerFactory");
                field.setAccessible(true);
                ch.qos.logback.classic.LoggerContext context = new ch.qos.logback.classic.LoggerContext();
                context.start();
                new ContextInitializer(context).autoConfig();
                field.set(binder, context);

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

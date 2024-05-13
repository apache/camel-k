package org.apache.maven.extensions.jsonlog;

import org.apache.maven.eventspy.AbstractEventSpy;
import org.apache.maven.rtinfo.RuntimeInformation;
import org.slf4j.LoggerFactory;

import javax.inject.Inject;
import javax.inject.Named;
import javax.inject.Singleton;
import java.lang.reflect.Field;

@Named
@Singleton
public class JsonLog extends AbstractEventSpy {

    RuntimeInformation runtimeInformation;

    @Inject
    JsonLog(RuntimeInformation runtimeInformation) throws Exception {
        this.runtimeInformation = runtimeInformation;

        try {
            // Maven 4 extension code:
            if (this.runtimeInformation.getMavenVersion().startsWith("4.")) {
                org.slf4j.spi.SLF4JServiceProvider provider = new ch.qos.logback.classic.spi.LogbackServiceProvider();
                provider.initialize();
                Field field = LoggerFactory.class.getDeclaredField("PROVIDER");
                field.setAccessible(true);
                field.set(null, provider);

                LoggerFactory.getLogger("json-logging").warn("Json Logging initialized");
            }
        } catch (Exception e) {
            e.printStackTrace();
            throw e;
        }
    }

}

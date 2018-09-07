package org.apache.camel.k.jvm;

import org.apache.camel.builder.RouteBuilder;
import org.apache.camel.main.Main;

import java.util.Objects;

public class Runner {

    public void run(String className) throws Exception {
        Objects.requireNonNull(className, "className must be present");

        Class<?> cls = Class.forName(className);
        Object instance = cls.newInstance();
        if (!RouteBuilder.class.isInstance(instance)) {
            throw new IllegalStateException("The class provided (" + className + ") is not a org.apache.camel.builder.RouteBuilder");
        }

        RouteBuilder builder = (RouteBuilder) instance;

        Main main = new Main();
        main.addRouteBuilder(builder);
        main.run();
    }

}

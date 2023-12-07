package org.apache.camel.k;

import io.quarkus.runtime.Quarkus;

public class Main {

    public static void main(String... args) {
        Quarkus.run(DepsFinder.class, args);
    }
}

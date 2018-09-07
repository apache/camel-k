package org.apache.camel.k.jvm;

public class Application {

    public static final String ENV_KAMEL_CLASS = "KAMEL_CLASS";

    public static void main(String[] args) throws Exception {

        String clsName = System.getenv(ENV_KAMEL_CLASS);
        if (clsName == null || clsName.trim().length() == 0) {
            throw new IllegalStateException("No valid class found in " + ENV_KAMEL_CLASS + " environment variable");
        }

        Runner runner = new Runner();
        runner.run(clsName);
    }

}

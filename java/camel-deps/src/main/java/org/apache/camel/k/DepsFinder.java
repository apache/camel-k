package org.apache.camel.k;

import java.util.HashSet;
import java.util.Locale;
import java.util.Set;
import java.util.TreeSet;
import java.util.Arrays;

import io.quarkus.runtime.QuarkusApplication;
import io.quarkus.runtime.annotations.QuarkusMain;
import org.apache.camel.main.KameletMain;
import org.apache.camel.main.download.DownloadListener;

@QuarkusMain
public class DepsFinder implements QuarkusApplication {

    private static String[] SKIP_DEPS = new String[]{"camel-core-languages", "camel-endpointdsl", "camel-java-joor-dsl"};

    @Override
    public int run(String... args) throws Exception {
        if (args.length == 0) {
            System.out.println("A file is required as an argument.");
            return 1;
        }
        String file = args[0];
        Locale.setDefault(Locale.US);

        final KameletMain main = new KameletMain();
//        main.setRepos(repos);
        main.setDownload(false);
        main.setFresh(false);
//        main.setMavenSettings(mavenSettings);
//        main.setMavenSettingsSecurity(mavenSettingsSecurity);
        DependencyListener depListener = new DependencyListener();
        main.setDownloadListener(depListener);
        main.setAppName("Apache Camel - Dependencies Discovery");

        main.setSilent(true);
        // enable stub in silent mode so we do not use real components
        main.setStubPattern("*");
        // do not run for very long in silent run
        main.addInitialProperty("camel.main.autoStartup", "false");
        main.addInitialProperty("camel.main.durationMaxSeconds", "1");
        main.addInitialProperty("camel.jbang.verbose", "false");
        main.addInitialProperty("camel.jbang.ignoreLoadingError", "true");
//        main.addInitialProperty("camel.component.kamelet.location", loc);
        main.addInitialProperty("camel.main.routesIncludePattern", "file:" + file);

        main.start();
        main.run();

        new TreeSet<>(depListener.getDependencies()).forEach(System.out::println);
        return 0;
    }

    private static class DependencyListener implements DownloadListener {
        final Set<String> deps = new HashSet<>();

        @Override
        public void onDownloadDependency(String groupId, String artifactId, String version) {
            if (Arrays.binarySearch(SKIP_DEPS, artifactId) < 0) {
                // add the camel-k dependency format for camel:<component-name>
                deps.add(artifactId.replace("camel-", "camel:"));
            }
        }

        @Override
        public void onAlreadyDownloadedDependency(String groupId, String artifactId, String version) {
        }

        Set<String> getDependencies() {
            return deps;
        }
    }
}

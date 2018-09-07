package org.apache.camel.k.jvm;

import org.junit.Ignore;
import org.junit.Test;

public class ApplicationTest {

    @Test
    @Ignore
    public void applicationTest() throws Exception {
        Runner runner = new Runner();
        runner.run(MyRoutes.class.getCanonicalName());
    }

}

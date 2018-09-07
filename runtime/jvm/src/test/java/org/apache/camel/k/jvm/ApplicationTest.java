package org.apache.camel.k.jvm;

import org.junit.Ignore;
import org.junit.Test;

public class ApplicationTest {
    @Test
    @Ignore
    public void applicationTest() throws Exception {
        Application.main(new String[] { MyRoutes.class.getCanonicalName() });
    }

}

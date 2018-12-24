package org.apache.camel.component.knative;

import org.apache.camel.Exchange;
import org.apache.camel.Processor;

/**
 * Converts objects prior to serializing them to external endpoints or channels
 */
public class KnativeConversionProcessor implements Processor {

    private boolean enabled;

    public KnativeConversionProcessor(boolean enabled) {
        this.enabled = enabled;
    }

    @Override
    public void process(Exchange exchange) throws Exception {
        if (enabled) {
            Object body = exchange.getIn().getBody();
            if (body != null) {
                byte[] newBody = Knative.MAPPER.writeValueAsBytes(body);
                exchange.getIn().setBody(newBody);
                exchange.getIn().setHeader("CE-ContentType", "application/json");
            }
        }
    }
}

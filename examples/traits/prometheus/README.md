# Camel K Prometheus Trait

In this section you will find examples about fine tuning your `Integration` using **Prometheus** `trait` capability.


A Prometheus-compatible endpoint is configured with the Prometheus trait. When utilising the Prometheus operator, it also generates a PodMonitor resource, which allows the endpoint to be scraped automatically.

To get statistics about the number of events successfully handled by the `Integration`,execute the `MyIntegration.java` route via:

    $ kamel run -t prometheus.enabled=true MyIntegration.java

 In case the prometheus operator is not installed in your cluster, run: 
    
    $ kamel run -t prometheus.enabled=true pod-monitor=false MyIntegration.java

You should be able to see the new integration running after a while via:

    $ kamel get 

The metrics can be retrieved by port-forwarding this service, e.g.:

    $ kubectl port-forward svc/metrics-prometheus 8080:8080

    $ curl http://localhost:8080/metrics

Similarly other use cases can be to retrieve information on unprocessed events, number of retries made to process an event, etc. For more information on Integration monitoring refer to the [Camel K Integration Monitoring](https://camel.apache.org/camel-k/next/observability/monitoring/integration.html) documentation.


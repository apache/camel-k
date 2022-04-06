# SERVICE EXAMPLE

This folder contains examples of how to use a `trait service`. You can use them to learn more about how to enable services for integrations deployed on the cluster.

To access integration outside the cluster you can enable a nodePort when you deploy integration. An example is `./RestDSL.java.`

You can also optionally decide to just go with the default clusterIP if you do not want your integration to be directly exposed to the outside world. An example of this use case is `./RestDSL2.java`
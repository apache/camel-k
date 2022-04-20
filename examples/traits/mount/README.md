# Camel K Mount Trait


In this section you will find examples about fine tuning your `Integration` using **Mount** `trait` capability.


**Example 1: Mount Persistent Volume Claim on the Integration Pod in a Kubernetes platform**

Create the Persistent Volume Claim:
`kubectl apply -f pvc-example.yaml`

Enable the Mount trait in the integrations, mount the Persistent Volume Claim, and log to an external volume:
`kamel run --trait mount.enabled=true --trait mount.volumes=["pvc-example:/tmp/log"] Producer.java -p quarkus.log.file.path=/tmp/log/example.log -p quarkus.log.file.enable=true`
`kamel run --trait mount.enabled=true --trait mount.volumes=["pvc-example:/tmp/log"] Consumer.java`
# Camel K Maven configuration examples

In this section, you'll find examples about Maven configurations in Camel K.

**Example 1: Reference Maven setting files in an IntegrationPlatform resource**

If you have a `settings.xml` file and (optionally) a `settings-security.xml` file, you can create a ConfigMap or a Secret for each file in Kubernetes:

`kubectl create configmap maven-settings --from-file=settings.xml`

`kubectl create configmap maven-settings-security --from-file=settings-security.xml`

The created ConfigMap(s) or Secret(s) can then be referenced in an IntegrationPlatform file like the example `ip.yaml`

With an IntegrationPlatform file, you can then create an IntegrationPlatform in Kubernetes:

`kubectl apply -f ip.yaml`
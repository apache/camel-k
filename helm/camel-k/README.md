# Camel K

Apache Camel K is the lightweight integration platform for Kubernetes: the easiest way to build and manage your Camel applications on Kubernetes. This chart deploys the Camel K operator and all resources needed to natively run Apache Camel Integrations on any Kubernetes cluster.

## Prerequisites

- A container image registry installed and configured for pull
- For production environments, a registry secret containing the access to container registry

### Minikube

Minikube offers a container registry addon, which it makes very well suited for local Camel K development and testing purposes:

```bash
$ minikube addons enable registry
```

You can use the container registry Service `registry` in namespace `kube-system` to configure in Camel K.

## Installation procedure

To install the chart, first add the Camel K repository:

```bash
$ helm repo add camel-k https://apache.github.io/camel-k/charts
```

## Install the operator

When installing the operator you must at least include the container registry to use (either the address or the service to use):

```bash
$ helm install camel-k camel-k/camel-k --set global=true \
  --set operator.env[0].name=REGISTRY_ADDRESS \
  --set operator.env[0].value=<my-registry-address> \
  --set operator.env[1].name=REGISTRY_SECRET \
  --set operator.env[1].value=<my-registry-secret>
```

In the case of a local registry available (for example, in Minikube):

```bash
$ helm install camel-k camel-k/camel-k --set global=true \
  --set operator.env[0].name=REGISTRY_SVC_NAMESPACE \
  --set operator.env[0].value=kube-system \
  --set operator.env[1].name=REGISTRY_SVC_NAME \
  --set operator.env[1].value=registry \
  --set operator.env[2].name=REGISTRY_INSECURE \
  --set-string operator.env[2].value=true
```

> **Note**: the installation RBAC provide the setting to access the Service in the namespace, you need to provide the specific RBAC if using another Service.

## Test your installation

Create a simple testing "Hello World" Integration as `test.yaml`:

```yaml
apiVersion: camel.apache.org/v1
kind: Integration
metadata:
  name: test
spec:
  flows:
  - from:
      parameters:
        period: "1000"
      steps:
      - setBody:
          simple: Hello Camel from ${routeId}
      - log: ${body}
      uri: timer:yaml
```

Run it on cloud:

```bash
kubectl apply -f test.yaml
```

Monitor how it is running:

```bash
$ kubectl get it -w
NAME   PHASE          READY   RUNTIME PROVIDER   RUNTIME VERSION   CATALOG VERSION   KIT                        REPLICAS
test   Building Kit           quarkus            3.8.1             3.8.1             kit-crc382j18sec73deq24g
test   Deploying              quarkus            3.8.1             3.8.1             kit-crc382j18sec73deq24g
test   Running        False   quarkus            3.8.1             3.8.1             kit-crc382j18sec73deq24g   0
test   Running        False   quarkus            3.8.1             3.8.1             kit-crc382j18sec73deq24g   1
test   Running        True    quarkus            3.8.1             3.8.1             kit-crc382j18sec73deq24g   1
```

For any problem, check it out the official [troubleshooting guide](https://camel.apache.org/camel-k/next/troubleshooting/troubleshooting.html) or the [documentation](https://camel.apache.org/camel-k/next/index.html).

## Additional installation time configuration

The [configuration](#configuration) section lists additional parameters that can be set during installation. From version 2.11.0 onward, the majority of parameters are expected to be configured via environment variables. See official documentation on Apache website for a full list.

> **Tip**: List all releases using `helm list`

## Upgrading the Chart

If you are upgrading the `camel-k` Deployment, you should always use a specific version of the chart and pre-install the CRDS:

```bash
# Upgrade the CRDs
$ curl -LO "https://github.com/apache/camel-k/raw/main/docs/charts/camel-k-x.y.z.tgz"
$ tar xvzf camel-k-x.y.z.tgz
$ kubectl replace -f camel-k/crds
# Upgrade the `camel-k` Deployment
$ helm upgrade camel-k/camel-k --version x.y.z
```

> **Note**: If you are using a custom ClusterRole instead of the default one `camel-k:edit` from `camel-k/crds/cluster-role.yaml` you should handle it appropriately.


## Uninstalling the Chart

To uninstall/delete the `camel-k` Deployment:

```bash
$ helm uninstall camel-k
```

The command removes all of the Kubernetes resources installed, except the CRDs.

To remove them:
```bash
$ curl -LO "https://github.com/apache/camel-k/raw/main/docs/charts/camel-k-x.y.z.tgz"
$ tar xvzf camel-k-x.y.z.tgz
$ kubectl delete -f camel-k/crds
```

## Configuration

The following table lists the most commonly configured parameters of the Camel K chart and their default values.

|           Parameter                    |             Description                                                   |            Default             |
|----------------------------------------|---------------------------------------------------------------------------|--------------------------------|
| `operator.operatorId`                  | The id of the Camel K operator                                            | `camel-k`                      |
| `operator.global`                      | Indicates if the operator should watch all namespaces                     | `false`                        |
| `operator.image`                       | The container image to use to run the operator                            | <the official image>           |
| `operator.nodeSelector`                | The nodeSelector to use for the operator                                  |                                |
| `operator.resources`                   | The resource requests and limits to use for the operator                  |                                |
| `operator.securityContext`             | The (container-related) securityContext to use for the operator           |                                |
| `operator.tolerations`                 | The list of tolerations to use for the operator                           |                                |
| `operator.imagePullSecret`             | The id of the Camel K operator                                            |                                |
| `operator.annotations`                 | The list of annotations to include to the operator Deployment             |                                |
| `operator.serviceAccount.annotations`  | The list of annotations to include to the operator Service Account        |                                |
| `extraEnv`                             | Extra env var on the operator Deployment (deprecated, use `env`)          |                                |
| `env`                                  | The operator configuration via environment variables                      |                                |

## Contributing

We'd like to hear your feedback and we love any kind of contribution!

The main contact points for the Camel K project are the [GitHub repository][1] and the [Camel K chat room][2].

[1]: https://github.com/apache/camel-k
[2]: https://camel.zulipchat.com

# Camel K

Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless superpowers: the easiest way to build and manage your Camel applications on Kubernetes. This chart deploys the Camel K operator and all resources needed to natively run Apache Camel Integrations on any Kubernetes cluster.

## Prerequisites

- Kubernetes 1.11+
- Container Image Registry installed and configured for pull

## Installation procedure

To install the chart, first add the Camel K repository:

```bash
$ helm repo add camel-k https://apache.github.io/camel-k/charts
```

## Install the operator

```bash
$ helm install camel-k camel-k/camel-k
```

## Set the container registry configuration

A regular installation requires you to provide a registry used by Camel K to build application containers. See official [Camel K registry documentation](https://camel.apache.org/camel-k/next/installation/registry/registry.html) or move to next section to run on a local Minikube cluster.

Create an `itp.yaml` file like:

```yaml
apiVersion: camel.apache.org/v1
kind: IntegrationPlatform
metadata:
  labels:
    app: camel-k
  name: camel-k
spec:
  build:
    registry:
      address: <my-registry-address>
      organization: <my-organization>
      secret: <my-secret-credentials>
```

and save the resource to the cluster with `kubectl apply -f itp.yaml`.

### Minikube

Minikube offers a container registry addon, which it makes very well suited for local Camel K development and testing purposes. You can see the cluster IP registry addon using the following script:

```bash
$ minikube addons enable registry
$ kubectl -n kube-system get service registry -o jsonpath='{.spec.clusterIP}'
```

Then you can provide the IntegrationPlatform as `itp.yaml`:

```yaml
apiVersion: camel.apache.org/v1
kind: IntegrationPlatform
metadata:
  labels:
    app: camel-k
  name: camel-k
spec:
  build:
    registry:
      address: <REGISTRY_ADDRESS>
      insecure: true
```

and save the resource to the cluster with `kubectl apply -f itp.yaml`.

## Test your installation

Verify the IntegrationPlatform is in Ready status:

```bash
kubectl get itp
NAME      PHASE   BUILD STRATEGY   PUBLISH STRATEGY   REGISTRY ADDRESS   DEFAULT RUNTIME
camel-k   Ready   routine          Jib                10.100.107.57      3.8.1
```

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

## Knative configuration

Camel K offers the possibility to run serverless Integrations in conjunction with [Knative operator](https://knative.dev). Once Knative and Camel K are installed on the same platform, you can configure Knative resources to be played by Camel K.

See instructions [how to enable Knative on Camel K](https://camel.apache.org/camel-k/next/installation/knative.html).

## Additional installation time configuration

The [configuration](#configuration) section lists additional parameters that can be set during installation.

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
| `operator.global`                      | Indicates if the operator should watch all namespaces                     | `false`                        |
| `operator.nodeSelector`                | The nodeSelector to use for the operator                                  |                                |
| `operator.resources`                   | The resource requests and limits to use for the operator                  |                                |
| `operator.securityContext`             | The (container-related) securityContext to use for the operator           |                                |
| `operator.tolerations`                 | The list of tolerations to use for the operator                           |                                |

## Contributing

We'd like to hear your feedback and we love any kind of contribution!

The main contact points for the Camel K project are the [GitHub repository][1] and the [Camel K chat room][2].

[1]: https://github.com/apache/camel-k
[2]: https://camel.zulipchat.com

# Camel K

Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless superpowers: the easiest way to build and manage your Camel applications on Kubernetes.

This chart deploys the Camel K operator and all resources needed to natively run Apache Camel integrations on any Kubernetes cluster.

## Prerequisites

- Kubernetes 1.11+
- Container Image Registry installed and configured for pull (optional in Openshift or Minikube)

## Installing the Chart

To install the chart, first add the Camel K repository:

```bash
$ helm repo add camel-k https://apache.github.io/camel-k/charts
```

Depending on the cloud platform of choice, you will need to specify a container registry at installation time.

### Plain Kubernetes

A regular installation requires you to provide a registry, used by Camel K to build application containers. See official [Camel K registry documentation](https://camel.apache.org/camel-k/next/installation/registry/registry.html).

```bash
$ helm install camel-k \
  --set platform.build.registry.address=<my-registry> \
  camel-k/camel-k
```

You may install Camel K and specify a container registry later.

### Openshift

If you are installing on OpenShift, Camel K can use the OpenShift internal registry to store and pull images:

```bash
$ helm install camel-k \
  --set platform.cluster=OpenShift \
  camel-k/camel-k
```

### Minikube

Minikube offers a container registry addon, which it makes very well suited for local Camel K development and testing purposes. You can export the cluster IP registry addon using the following script:

```bash
$ minikube addons enable registry
$ export REGISTRY_ADDRESS=$(kubectl -n kube-system get service registry -o jsonpath='{.spec.clusterIP}')
```

Then you can install Camel K with:

```bash
$ helm install camel-k \
  --set platform.build.registry.address=${REGISTRY_ADDRESS} \
  --set platform.build.registry.insecure=true \
  camel-k/camel-k
```

### Knative configuration

Camel K offers the possibility to run serverless Integrations in conjunction with [Knative operator](https://knative.dev). Once Knative and Camel K are installed on the same platform, you can configure Knative resources to be played by Camel K.

See instructions [how to enable Knative on Camel K](https://camel.apache.org/camel-k/next/installation/knative.html).

### Additional installation time configuration

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

The following table lists the most commonly configured parameters of the Camel K chart and their default values. The chart allows configuration of an `IntegrationPlatform` resource, which among others includes build properties and traits configuration. A full list of parameters can be found [in the operator specification][1].

|           Parameter                    |             Description                                                   |            Default             |
|----------------------------------------|---------------------------------------------------------------------------|--------------------------------|
| `platform.build.registry.address`      | The address of a container image registry to push images                  |                                |
| `platform.build.registry.secret`       | A secret used to push/pull images to the Docker registry                  |                                |
| `platform.build.registry.organization` | An organization on the Docker registry that can be used to publish images |                                |
| `platform.build.registry.insecure`     | Indicates if the registry is not secured                                  | true                           |
| `platform.cluster`                     | The kind of Kubernetes cluster (Kubernetes or OpenShift)                  | `Kubernetes`                   |
| `platform.profile`                     | The trait profile to use (Knative, Kubernetes or OpenShift)               | auto                           |
| `operator.global`                      | Indicates if the operator should watch all namespaces                     | `false`                        |
| `operator.resources`                   | The resource requests and limits to use for the operator                  |                                |
| `operator.securityContext`             | The (container-related) securityContext to use for the operator           |                                |
| `operator.tolerations`                 | The list of tolerations to use for the operator                           |                                |

## Contributing

We'd like to hear your feedback and we love any kind of contribution!

The main contact points for the Camel K project are the [GitHub repository][2] and the [Camel K chat room][3].

[1]: https://camel.apache.org/camel-k/next/architecture/cr/integration-platform.html
[2]: https://github.com/apache/camel-k
[3]: https://camel.zulipchat.com

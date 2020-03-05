# Camel K

Apache Camel K is a lightweight integration platform, born on Kubernetes,
with serverless superpowers.

This chart deploys the Camel K operator and all resources needed to natively run
Apache Camel integrations on any Kubernetes cluster.

## Prerequisites

- Kubernetes 1.11+
- Container Image Registry installed and configured for pull

## Installing the Chart

To install the chart, first add the Camel K repository:

```bash
$ helm repo add camel-k https://apache.github.io/camel-k/charts
```

If you are installing on OpenShift, Camel K can use the OpenShift internal registry to
store and pull images.

Installation on OpenShift can be done with command:

```bash
$ helm install \
  --generate-name \
  --set platform.cluster=OpenShift \
  camel-k/camel-k
```

When running on a cluster with no embedded internal registry, you need to specify the address
and properties of an image registry that the cluster can use to store image.

For example, on Minikube you can enable the internal registry and get its address:

```bash
$ minikube addons enable registry
$ export REGISTRY_ADDRESS=$(kubectl -n kube-system get service registry -o jsonpath='{.spec.clusterIP}')
```

Then you can install Camel K with:

```bash
$ helm install \
  --generate-name \
  --set platform.build.registry.address=${REGISTRY_ADDRESS} \
  --set platform.build.registry.insecure=true \
  camel-k/camel-k
```

The [configuration](#configuration) section lists
additional parameters that can be set during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `camel-k` Deployment:

```bash
$ helm delete camel-k
```

The command removes all the Kubernetes resources installed.

## Configuration

The following table lists the most commonly configured parameters of the
CouchDB chart and their default values:

|           Parameter                |             Description                                     |                Default                 |
|------------------------------------|-------------------------------------------------------------|----------------------------------------|
| `platform.build.registry.address`  | The address of a container image registry to push images    |                                        |
| `platform.build.registry.insecure` | Indicates if the registry is secured                        | true                                   |
| `platform.cluster`                 | The kind of Kubernetes cluster (Kubernetes or OpenShift)    | `Kubernetes`                           |
| `platform.profile`                 | The trait profile to use (Knative, Kubernetes or OpenShift) | auto                                   |

## Contributing

We'd like to hear your feedback and we love any kind of contribution!

The main contact points for the Camel K project are the [GitHub repository][1]
and the [Gitter room][2].

[1]: https://github.com/apache/camel-k
[2]: https://gitter.im/apache/camel-k

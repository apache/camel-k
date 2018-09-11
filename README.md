# Apache Camel K

Apache Camel K (a.k.a. Kamel) is a lightweight integration framework built from Apache Camel that runs natively on Kubernetes and is specifically designed for serverless and microservice architectures.

## Getting Started

You can run Camel K integrations on a Kubernetes or Openshift cluster, so you can choose to create a development cluster or use a cloud instance
for Camel K.

### Creating a Development Cluster
There are various options for creating a development cluster:

**Minishift**

You can run Camel K integrations on Openshift using the Minishift cluster creation tool.
Follow the instructions in the [getting started guide](https://github.com/minishift/minishift#getting-started) for the installation.

After installing the `minishift` binary, you need to enable the `admin-user` addon:

```
minishift addons enable admin-user
```

Then you can start the cluster with:

```
minishift start
```

**Minikube**

Minikube and Kubernetes are not yet supported (but support is coming soon).

### Setting Up the Cluster

To start using Camel K you need the **"kamel"** binary, that can be used to both configure the cluster and run integrations.

There's currently no release channel for the "kamel" binary, so you need to **build it from source!** Refer to the [building section](#building)
for information on how to do it.

Once you have the "kamel" binary, log into your cluster using the "oc" or "kubectl" tool and execute the following command to install Camel K:

```
kamel install
```

This will configure the cluster with the Camel K custom resource definitions and install the operator on the current namespace.

**Note:** Custom Resource Definitions (CRD) are cluster-wide objects and you need admin rights to install them. Fortunately this
operation can be done once per cluster. So, if the `kamel install` operation fails, you'll be asked to repeat it when logged as admin.
For Minishift, this means executing `oc login -u system:admin` then `kamel install --cluster-setup` only for first-time installation.

### Running a Integration

After the initial setup, you can run a Camel integration on the cluster executing:

```
kamel run runtime/examples/Sample.java
```

A "Sample.java" file is included in the folder runtime/examples of this repository. You can change the content of the file and execute the command again to see the changes.

A JavaScript integration has also been provided as example, to run it:

```
kamel run runtime/examples/routes.js
```

### Monitoring the Status

Camel K integrations follow a lifecycle composed of several steps before getting into the `Running` state.
You can check the status of all integrations by executing the following command:

```
kamel get
```

### Uninstalling Camel K

If requires to uninstall Camel K from the OpenShift or Kubernetes, it's nessesary to run following command using "oc" or "kubectl" tool:

```
delete all,pvc,configmap,rolebindings,clusterrolebindings,secrets,sa,roles,clusterroles,crd -l 'app=camel-k'
```

## Building

In order to build the project follow these steps:
- this project is supposed to be cloned in `$GOPATH/src/github.com/apache/camel-k`
- install dep: https://github.com/golang/dep The last version is 0.5.0 and it's requested to use this version to be able to be aligned on each build.
- install operator-sdk: https://github.com/operator-framework/operator-sdk
- dep ensure -v
- make build

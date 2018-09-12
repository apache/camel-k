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

There's currently no release channel for the "kamel" binary, so you need to **build it from source!** Refer to the [contributing guide](#contributing)
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

## Contributing

We love contributions!

The project is written in [Go](https://golang.org/) and contains some parts written in Java for the [integration runtime](/runtime).
Camel K is built on top of Kubernetes through *Custom Resource Definitions*. The [Operator SDK](https://github.com/operator-framework/operator-sdk) is used
to manage the lifecycle of those custom resources.

### Requirements

In order to build the project, you need to comply with the following requirements:
- **Go version 1.10+**: needed to compile and test the project. Refer to the [Go website](https://golang.org/) for the installation.
- **Dep version 0.5.0**: for managing dependencies. You can find installation instructions in the [dep GitHub repository](https://github.com/golang/dep).
- **Operator SDK v0.0.6+**: used to build the operator and the Docker images. Instructions in the [Operator SDK website](https://github.com/operator-framework/operator-sdk).
- **GNU Make**: used to define composite build actions. This should be already installed or available as package if you have a good OS (https://www.gnu.org/software/make/).

### Checking Out the Sources

You can create a fork of this project from Github, then clone your fork with the `git` command line tool.

You need to put the project in your $GOPATH (refer to [Go documentation](https://golang.org/doc/install) for information).
So, make sure that the **root** of the github repo is in the path:

```
$GOPATH/src/github.com/apache/camel-k/
```

### Structure

This is a high level overview of the project structure:

- [/cmd](/cmd): contains the entry points (the *main* functions) for the **camel-k-operator** binary and the **kamel** client tool.
- [/build](/build): contains scripts used during make operations for building the project.
- [/deploy](/deploy): contains Kubernetes resource files that are used by the **kamel** client during installation. The `/deploy/resources.go` file is kept in sync with the content of the directory (`make build-embed-resources`), so that resources can be used from within the go code.
- [/pkg](/pkg): this is where the code resides. The code is divided in multiple subpackages.
- [/runtime](/runtime): the Java runtime code that is used inside the integration Docker containers.
- [/tmp](/tmp): scripts and Docker configuration files used by the operator-sdk.
- [/vendor](/vendor): project dependencies.
- [/version](/version): contains the global version of the project.

### Building

Go dependencies in the *vendor* directory are not included when you clone the project.

Before compiling the source code, you need to sync your local *vendor* directory with the project dependencies, using the following command:

```
make dep
```

The `make dep` command runs `dep ensure -v` under the hood, so make sure that `dep` is properly installed.

To build the whole project you now need to run:

```
make
```

This execute a full build of both the Java and Go code. If you need to build the components separately you can execute:
- `make build-operator`: to build the operator binary only.
- `make build-kamel`: to build the `kamel` client tool only.
- `make build-runtime`: to build the Java-based runtime code only.

After a successful build, if you're connected to a Docker daemon, you can build the operator Docker image by running:

```
make images
```

### Testing

Unit tests are executed automatically as part of the build. They use the standard go testing framework.

Integration tests (aimed at ensuring that the code integrates correctly with Kubernetes and Openshift), need special care.

The **convention** used in this repo is to name unit tests `xxx_test.go`, and name integration tests `yyy_integration_test.go`.

Since both names end with `_test.go`, both would be executed by go during build, so you need to put a special **build tag** to mark
integration tests. A integration test should start with the following line:

```
// +build integration
```

An [example is provided here](https://github.com/apache/camel-k/blob/ff672fbf54c358fca970da6c59df378c8535d4d8/pkg/build/build_manager_integration_test.go#L1).

Before running a integration test, you need to:
- Login to a Kubernetes/Openshift cluster.
- Set the `KUBERNETES_CONFIG` environment variable to point to your Kubernetes configuration file (usually `<home-dir>/.kube/config`).
- Set the `WATCH_NAMESPACE` environment variable to a Kubernetes namespace you have access to.
- Set the `OPERATOR_NAME` environment variable to `camel-k-operator`.

When the configuration is done, you can run the following command to execute **all** integration tests:

```
make test-integration
```

### Running

If you want to install everything you have in your source code and see it running on Kubernetes, you need to follow these steps:
- `make`: to build the project.
- `eval $(minishift docker-env)`: to connect to your Minishift Docker daemon.
- `make images`: to build the operator docker image.
- `./kamel install`: to install Camel K into the namespace.
- `oc delete pod -l name=camel-k-operator`: to ensure the operator is using latest image (delete the pod to let Openshift recreate it).

**Note for contributors:** why don't you embed all those steps in a `make install-minishift` command?

Now you can play with Camel K:

```
./kamel run runtime/examples/Sample.java
```

To add additional dependencies to your routes: 

```
./kamel run -d camel:dns runtime/examples/dns.js
```


### Debugging and Running from IDE

Sometimes it's useful to debug the code from the IDE when troubleshooting.

**Debugging the `kamel` binary**

It should be straightforward: just execute the [/cmd/kamel/kamel.go]([/cmd/kamel/kamel.go]) file from the IDE (e.g. Goland) in debug mode.

**Debugging the operator**

It is a bit more complex (but not so much).

You are going to run the operator code **outside** Openshift in your IDE so, first of all, you need to **stop the operator running inside**:

```
oc scale deployment/camel-k-operator --replicas 0
```

You can scale it back to 1 when you're done and you have updated the operator image.

You can setup the IDE (e.g. Goland) to execute the [/cmd/camel-k-operator/camel_k_operator.go]([/cmd/camel-k-operator/camel_k_operator.go]) file in debug mode.

When configuring the IDE task, make sure to add all required environment variables in the *IDE task configuration screen* (such as `KUBERNETES_CONFIG`, as explained in the [testing](#testing) section).

## Uninstalling Camel K

If required, it is possible to completely uninstall Camel K from OpenShift or Kubernetes with the following command, using the "oc" or "kubectl" tool:

```
# kubectl if using kubernetes
oc delete all,pvc,configmap,rolebindings,clusterrolebindings,secrets,sa,roles,clusterroles,crd -l 'app=camel-k'
```

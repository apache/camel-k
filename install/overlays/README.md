
# Kustomize Camel K

Kustomize provides a declarative approach to the configuration customization of a Camel-K installation. Kustomize works either with a standalone executable or as a built-in to kubectl.

Basic overlays are provided for easy usage.

## HOW-TO

### Initialize

First create a new kustomization from the wanted version (kubernetes or openshift) in the repository:
```sh
kustomize create --resources https://github.com/apache/camel-k.git/install/overlays/kubernetes\?ref\=exp/kustomize_structure
```

You can also clone the camel-k repository and reference the local folder :
```sh
kubectl kustomize <path/to/localrepo/install/overlays/openshift | kubectl create -f -
```

To ensure the `IntegrationPlatform` custom resource is created add in the `kustomization.yaml`:
```yaml
sortOptions:
  order: fifo
```

### Configuration

Camel K Operators offers several possibility of customization. The default installation needs to but cutomized in most of the cases, but, we have a series of configuration that can be applied when you want to fine tune your Camel K operator and get the very best of it.


#### Operator configuration

TODO: test knative
TODO: add namespace adaptation
TODO: find scorecard usage

The operator installation can be customized by using the following parameters:

* Set the operator id that is used to select the resources this operator should manage (default "camel-k") (see `install/overlays/common/patches/patch-operator-id-deployment.yaml` and `install/overlays/common/patches/patch-operator-id-integration-platform.yaml`)
* Set the operator Image used for the operator deployment (using https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#field-name-images)
* Set the operator ImagePullPolicy used for the operator deployment (see `config/manager/patch-image-pull-policy-always.yaml`)

#### Resources managment

We provide certain configuration to better "operationalize" the Camel K Operator:

* Add a NodeSelector to the operator Pod (see `config/manager/patch-node-selector.yaml`)
* Define the resources requests and limits assigned to the operator Pod as <requestType.requestResource=value> (i.e., limits.memory=256Mi) (see `config/manager/patch-resource-requirement.yaml`)
* Add a Toleration to the operator Pod (see `config/manager/patch-toleration.yaml`)

#### Build configuration

We have several configuration used to influence the building of an integration  (see `install/overlays/common/patches/patch-build-integration-platform.yaml`):

* Set the base Image used to run integrations
* Set the build publish strategy
* Add a build publish strategy option, as <name=value>
* Set the build strategy
* Set the build order strategy
* Set how long the build process can last
* Set how long the catalogtool image build can last


A very important set of configuration you can provide is related to Maven (see `install/overlays/common/patches/patch-maven-integration-platform.yaml`):

* Configure the secret key containing the Maven CA certificates (secret/key)
* Add a default Maven CLI option to the list of arguments for Maven commands
* Add a Maven build extension
* Path of the local Maven repository
* Add a Maven property
* Configure the source of the Maven settings (configmap|secret:name[/key])

#### Publish configuration

Camel K requires a container registry where to store the applications built (see `install/overlays/common/patches/patch-registry-integration-platform.yaml`). These are the main configurations:

* A organization on the Docker Hub that can be used to publish images
* A container registry that can be used to publish images
* Configure registry access in insecure mode or not (`http` vs `https`)
* A secret used to push/pull images to the container registry containing authorization tokens for pushing and pulling images

#### Monitoring

Camel K Operator provides certain monitoring capabilities.

You can activate the monitoring by adding the following resources: `config/prometheus`

You can change the default settings:
* The port of the health endpoint (default 8081) (see `config/manager/patch-toleration.yaml`)
* The port of the metrics endpoint (default 8080) (see `config/manager/patch-toleration.yaml`)
* The level of operator logging (default - info): info or 0, debug or 1 (default "info") (see `config/manager/patch-log-level.yaml`)


#### Installation Topology

By default the proposed overlays configure the cluster, install an integration platform and the operator. You can easilly build your own overlay with only part or the configuration to fit your need.

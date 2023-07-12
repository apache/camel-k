# Kamel run

This Task creates and run a [Camel K](https://github.com/apache/camel-k) Integration.

If you have installed Camel K operator, you can configure the `kamel-run` task the step to create the Integration that will be operated by Camel K. You can configure the parameter to delegate the build of your application to Camel K or do it as a part of previous pipelines steps.

## Install the Task

```shell
kubectl apply -f https://raw.githubusercontent.com/apache/camel-k/main/tekton/kamel-run/0.1/kamel-run.yaml
```

## Parameters

- **camel-k-image**: The name of the image containing the Kamel CLI (_default:_ docker.io/apache/camel-k:1.12.0).
- **filename**: the file containing the Integration source.
- **namespace**: the namespace where to run the Integration (_default:_ the task execution namespace).
- **container-image**: the container image to use for this Integration. Useful when you want to build your own container for the Integration (_default:_ empty, will trigger an Integration build).
- **wait**: wait for the Integration to run before finishing the task. Useful when you want to get the **integration-phase** result (_default:_ "false").

## Workspaces

* **source**: A [Workspace](https://github.com/tektoncd/pipeline/blob/main/docs/workspaces.md) containing the Integration source to run.

## Results

- **integration-name**: the Integration name which was created/updated.
- **integration-phase**: the status of the Integration, tipycally used with **wait: true** input parameter.

## Platforms

The Task can be run on `linux/amd64` platform.

## Usage

The Task can be used in several ways to accomodate the different build and deployment strategy you may have.

### Create the Service Account

As we will do delegate the task, the creation of an Integration, we need to provide a `ServiceAccount` with the privileges required by the tasks:

```shell
kubectl apply -f  https://raw.githubusercontent.com/apache/camel-k/main/tekton/kamel-run/0.1/support/camel-k-tekton.yaml
```

### Delegate build to operator

Use the [Tekton Camel K operator builder sample](../0.1/samples/run-operator-build.yaml) in order to fetch a Git repository and run a Camel K Integration delegating the build to the Camel K operator.

### Full pipeline with custom build

Use the [Tekton Camel K external builder sample](../0.1/samples/run-external-build.yaml) as a reference for a full pipeline where you define your own process of building the Camel application and using the `kamel-run` Task as last step in order to deploy the Integration and let Camel K operator managing it.
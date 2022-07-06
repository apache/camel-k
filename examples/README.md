# Camel K Examples

This folder contains various examples of `Camel K` integrations. You can use them to learn more about the capabilities of Camel K or to inspire your integration development.

## Basic usage examples

In this section you will find the most basic examples. Useful to start learning about Camel K and how to run. You can use many supported languages and learn about the most basic features:

| Type  |  Description | Link  |
|---|---|---|
| Languages | Simple integrations developed in various supported languages | [see examples](./languages/)|
| User Config | Explore how to include a `property`, `secret`, `configmap` or file `resource` in your integration | [see examples](./user-config/)|
| User Dependencies | Explore how to include a local dependency in your integration with Jitpack | [see examples](./jitpack/)|
| Processor | Show how to include `Processor`s logic | [see examples](./processor/)|
| Open API | `Open API` support | [see examples](./openapi/)|
| Rest | Produce/Consume `REST`ful services | [see examples](./rest/)|
| Modeline | [Camel K modeline support](https://camel.apache.org/camel-k/latest/cli/modeline.html) | [see examples](./modeline/)|
| Volumes | Produce/Consume files attached to a `PVC` | [see examples](./volumes/)|

## Component usage examples

In this section you can find a few examples of certain [`Camel` components](https://camel.apache.org/components/latest/index.html). This is a limited number of the wide variety of components supported by Apache Camel. You can also find useful examples [in this repository](https://github.com/apache/camel-k-examples).

| Type  |  Description | Link  |
|---|---|---|
| HTTP/HTTPS | Component usage | [see examples](./http/)|
| Kafka | Component usage | [see examples](./kafka/)|
| Knative | Component usage | [see examples](./knative/)|

## Advanced usage examples

As soon as you will learn the basic stuff, you will like to try the new advanced feature offered by Camel K. Here a few examples:

| Type  |  Description | Link  |
|---|---|---|
| Kamelets | How to use [`Kamelet`s](https://camel.apache.org/camel-k/latest/kamelets/kamelets.html) | [see examples](./kamelets/)|
| Master | Master support example | [see examples](./master/)|
| OLM | OPERATOR Lifecycle manager installation example | [see examples](./olm/)|
| Polyglot | Polyglot integration examples | [see examples](./polyglot/)|
| Pulsar | Pulsar usage | [see examples](./pulsar/)|
| Saga | Saga pattern example | [see examples](./saga/)|
| Tekton | Tekton tutorial | [see examples](./tekton/)|

## Traits usage examples

Traits configuration will be very helpful to fine tune your `Integration`. Here a few examples:

| Type  |  Description | Link  |
|---|---|---|
| Container | How to customize with `container` trait| [see examples](./traits/container/)|
| JVM | How to use `jvm` trait| [see examples](./traits/jvm/)|
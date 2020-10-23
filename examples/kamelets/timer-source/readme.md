# Timer Source Hello World example

This example shows how to create a simple timer-source `kamelet` and how to use it in a new integration or how to bind it to a knative destination through a `KameletBinding`.

## Timer Source Kamelet

First of all, you must install the timer source kamelet defined in `timer-source.kamelet.yaml` file:
```
$ kubectl apply -f timer-source.kamelet.yaml
```
You can check the newly created `kamelet` checking the list of kamelets available:
```
$ kubectl get kamelets

NAME                   PHASE
timer-source           Ready
```
## Timer Source integration
As soon as the `kamelet` is available in your cluster, you can use it in any integration such as the one defined in `usage.groovy` file:
```
from('kamelet:timer-source?message=Hello+Kamelets&period=1000')
    .log('${body}')
```
Just run the integration via:
```
$ kamel run usage.groovy
```
You should be able to see the new integration running after some time:
```
$ kamel get
NAME	PHASE	KIT
usage	Running	kit-bu9d2r22hhmoa6qrtc2g
```
As soon as the integration starts, you will be able to log the timer source events emitted:
```
$ kamel log usage

[1] Monitoring pod usage-785d65897b-5jdvp
...
[1] 2020-10-23 12:51:19,724 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
[1] 2020-10-23 12:51:20,681 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
[1] 2020-10-23 12:51:21,682 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
[1] 2020-10-23 12:51:22,681 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
[1] 2020-10-23 12:51:23,684 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
[1] 2020-10-23 12:51:24,682 INFO  [route1] (Camel (camel-1) thread #0 - timer://tick) Hello Kamelets
....
```
## Timer Source KameletBinding
You can also bind the `kamelet` to a knative destination (or other events channel, [see the official documention](https://camel.apache.org/camel-k/latest/kamelets/kamelets.html#kamelets-usage-binding)) in order to source the events emitted by the configuration described by the `kamelet`. Make sure to have Knative properly installed on your cluster ([see installation guide](https://knative.dev/docs/install/)).

First of all, you must declare the knative destination:
```
kubectl apply -f messages-channel.yaml
```
Once the destination is ready, you can reference it from a `KameletBinding` such as in the example `kamelet-binding-example.yaml`. In order to bind the connector to the destination apply the configuration:
```
$ kubectl apply -f kamelet-binding-example.yaml
```
You can confirm the creation of the `KameletBinding` listing the resources:
```
$ kubectl get kameletbindings

NAME           PHASE
timer-source   Ready
```
At this stage you will be able to consume the timer event sources from the knative destination.

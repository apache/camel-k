# Timer Source to Log Sink

This example shows how to create a simple timer `event source` and a log `event sink`. The timer events emitted are consumed by a simple logging connector which will print out those events.

## Create events source and sink

Let's start by creating the timer event source and log event sink as `kamelet`s.
```
$ kubectl apply -f timer-source.kamelet.yaml
$ kubectl apply -f log-sink.kamelet.yaml
```

You can check the newly created `kamelet`s in the list.
```
$ kubectl get kamelets

NAME           PHASE
log-sink       Ready
timer-source   Ready
```

## Create channel destination

Let's continue by creating a `knative` destination.
```
$ kubectl apply -f timer-events.yaml
```

## Binding events

We can now bind the timer event source to produce events on the destination with the `timer-source.binding.yaml` configuration.
```
$ kubectl apply -f timer-source.binding.yaml
```
In a similar fashion you can bind to the log sink in order to consume those events with the `log-sink.binding.yaml` configuration.
```
$ kubectl apply -f log-sink.binding.yaml
```
You can check the newly created bindings listing the `KameletBidings`.
```
$ kubectl get KameletBindings

NAME                 PHASE
log-event-sink       Ready
timer-event-source   Ready
```

### Watch the event sink

After a while you will be able to watch the event consumed by the underlying `log-event-sink` integration:

```
$ kamel log log-event-sink

[1] Monitoring pod log-event-sink-wjm9w-deployment-cf4f49655-xwq82
...
[1] 2020-10-23 14:28:11,878 INFO  [sink] (vert.x-worker-thread-1) Exchange[ExchangePattern: InOnly, BodyType: byte[], Body: Hello world!]
[1] 2020-10-23 14:28:11,877 INFO  [sink] (vert.x-worker-thread-0) Exchange[ExchangePattern: InOnly, BodyType: byte[], Body: Hello world!]
[1] 2020-10-23 14:28:12,381 INFO  [sink] (vert.x-worker-thread-2) Exchange[ExchangePattern: InOnly, BodyType: byte[], Body: Hello world!]
[1] 2020-10-23 14:28:13,276 INFO  [sink] (vert.x-worker-thread-3) Exchange[ExchangePattern: InOnly, BodyType: byte[], Body: Hello world!]
[1] 2020-10-23 14:28:14,299 INFO  [sink] (vert.x-worker-thread-4) Exchange[ExchangePattern: InOnly, BodyType: byte[], Body: Hello world!]
```
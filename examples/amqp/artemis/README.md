# How to deploy a simple JMS/AMQP Broker to Kubernetes cluster

This is a very simple example to show how to create a JMS/AMQP broker. **Note**, this is not ready for any production purpose.

## Create a Kubernetes Deployment

You can [install ActiveMQ Artemis on Kubernetes](https://artemiscloud.io/blog/using_operator/) thanks to ArtemisCloud.io. It would be enough to execute step 1, 2 and 3 of the linked blog post.

## Create an AMQP broker instance

Once the operator is up and running, you can proceed by creating a basic install of a broker accepting `amqp` protocol:

```
kubectl apply -f artemis-amqp.yaml
```

## Expose the Broker via service

We need to make the broker accessible from the `Integration`. Let's do that via a `Service`:

```
kubectl apply -f artemis-service.yaml

```

We're now ready to use the `Broker`.

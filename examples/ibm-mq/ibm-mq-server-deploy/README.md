# How to deploy a IBM MQ Server to Kubernetes cluster

This is a very simple example to show how to install an IBM MQ Server. **Note**, this is not ready for any production purpose.

The Deployment uses IBM MQ Server image from docker hub.

## Install the IBM MQ Server
```
kubectl create -f ibm-mq-server.yaml
```

This will create a server with the following data:

```
App User     : app
App Password : ibmmqpasswd
Queue manager: QM1
Queue        : DEV.QUEUE.1
Channel      : DEV.APP.SVRCONN
```

## Remove the IBM MQ Server resources

```
kubectl delete -f ibm-mq-server.yaml
```

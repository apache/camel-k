# OPERATOR Lifecycle manager (OLM) installation 

Camel K supports as default an installation procedure in order to get all the features offered by [OLM](https://olm.operatorframework.io/). We have in place a default mechanism to discover the presence of OLM installed on your cluster.

NOTE: You can disable the feature by providing the `--olm=false` during the installation procedure.

## Install on Minikube

An interesting way to test locally the OLM is to use the **Minikube OLM addon**. If you have a local Minikube, you can proceed with the following steps to install the OLM:

```
minikube addons enable olm
```

As soon as all the resources are installed, you can now proceed by installing Camel K operator as you used to do, ie:

```
kamel install --global
```

In order to verify the installation, you can check the `ClusterServiceVersion` (CSV) custom resource:

```
kubectl get csv
NAME                      DISPLAY            VERSION   REPLACES                  PHASE
camel-k-operator.v1.8.2   Camel K Operator   1.8.2     camel-k-operator.v1.8.1   Succeeded
```

You can now run any integration as you used to do.
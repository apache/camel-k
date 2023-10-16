# Kubernetes overlay

## Pre-requise

This is an overlay intended for Minkube with the following configuration:
* Cluster-admin privileges are required
* Namespace is `default`
* Operator id is `camel-k`
* An available registry

## Usage

The following env variable are expected


To run from local folder :
```sh
kubectl kustomize kustomize/overlays/kubernetes | kubectl create -f -
```

To run from remote github repository:
```sh
kubectl kustomize https://github.com/apache/camel-k/kustomize/overlays/kubernetes | kubectl create -f -
```

NOTE: to use a different branch add the parameter "ref" to the github repository URL.


### Minikube

You can easilly configure minikube with the registry addon.

First get the internal registry service IP from minikube :

```sh
export KAMEL_REGISTRY_ADDRESS="$(kubectl get service --selector "kubernetes.io/minikube-addons"="registry" --namespace kube-system -o=jsonpath='{.items[0].spec.clusterIP}')"
```

Then patch registry:
```yaml
- op: replace
  path: /spec/build/registry
  value:
    insecure: true
    address: ${KAMEL_REGISTRY_ADDRESS}
```

Finally run your modified version:

kubectl kustomize . | envsubst '$KAMEL_REGISTRY_ADDRESS' | kubectl create -f -
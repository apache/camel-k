## Openshift overlay

## Pre-requise

This is an overlay intended for Openshift without OLM with the following configuration:
* Cluster-admin privileges are required
* Namespace is `default`
* Operator id is `camel-k`

## Usage

To run from local folder :
```sh
kubectl kustomize install/overlays/openshift | kubectl create -f -
```

To run from remote github repository:
```sh
kubectl kustomize https://github.com/apache/camel-k/install/overlays/openshift kubectl create -f -
```

NOTE: to use a different branch add the parameter "ref" to the github repository URL.

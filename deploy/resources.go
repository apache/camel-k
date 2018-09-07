/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by build/embed_resources.sh. DO NOT EDIT.

package deploy

var Resources map[string]string

func init() {
	Resources = make(map[string]string)

	Resources["crd-integration-context.yaml"] =
		`
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: integrationcontexts.camel.apache.org
  labels:
    app: "camel-k"
spec:
  group: camel.apache.org
  names:
    kind: IntegrationContext
    listKind: IntegrationContextList
    plural: integrationcontexts
    singular: integrationcontext
  scope: Namespaced
  version: v1alpha1

`
	Resources["crd-integration.yaml"] =
		`
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: integrations.camel.apache.org
  labels:
    app: "camel-k"
spec:
  group: camel.apache.org
  names:
    kind: Integration
    listKind: IntegrationList
    plural: integrations
    singular: integration
  scope: Namespaced
  version: v1alpha1

`
	Resources["cr.yaml"] =
		`
apiVersion: "camel.apache.org/v1alpha1"
kind: "Integration"
metadata:
  name: "example"
spec:
  replicas: 1
  source:
    code: |-
      package kamel;

      import org.apache.camel.builder.RouteBuilder;

      public class Routes extends RouteBuilder {

          @Override
          public void configure() throws Exception {
              from("timer:tick")
                .setBody(constant("Hello World!!!"))
                .to("log:info");
          }

      }

`
	Resources["operator-deployment.yaml"] =
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"
spec:
  replicas: 1
  selector:
    matchLabels:
      name: camel-k-operator
  template:
    metadata:
      labels:
        name: camel-k-operator
    spec:
      serviceAccountName: camel-k-operator
      containers:
        - name: camel-k-operator
          image: docker.io/apache/camel-k:0.0.1-SNAPSHOT
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - camel-k-operator
          imagePullPolicy: IfNotPresent
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "camel-k-operator"

`
	Resources["operator-role-binding.yaml"] =
		`
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"
subjects:
- kind: ServiceAccount
  name: camel-k-operator
roleRef:
  kind: Role
  name: camel-k-operator
  apiGroup: rbac.authorization.k8s.io

`
	Resources["operator-role-kubernetes.yaml"] =
		`
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"
rules:
- apiGroups:
  - camel.apache.org
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  attributeRestrictions: null
  resources:
  - daemonsets
  verbs:
  - get
  - list
  - watch

`
	Resources["operator-role-openshift.yaml"] =
		`
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"
rules:
- apiGroups:
  - camel.apache.org
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  attributeRestrictions: null
  resources:
  - daemonsets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  - "build.openshift.io"
  resources:
  - buildconfigs
  - buildconfigs/webhooks
  - builds
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  - "image.openshift.io"
  resources:
  - imagestreamimages
  - imagestreammappings
  - imagestreams
  - imagestreams/secrets
  - imagestreamtags
  verbs:
  - create
  - delete
  - deletecollection
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  - build.openshift.io
  attributeRestrictions: null
  resources:
  - buildconfigs/instantiate
  - buildconfigs/instantiatebinary
  - builds/clone
  verbs:
  - create

`
	Resources["operator-service-account.yaml"] =
		`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"

`
	Resources["operator-service.yaml"] =
		`
apiVersion: v1
kind: Service
metadata:
  labels:
    name: camel-k-operator
    app: "camel-k"
  name: camel-k-operator
spec:
  ports:
    - name: metrics
      port: 60000
      protocol: TCP
      targetPort: metrics
  selector:
    name: camel-k-operator
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}

`
	Resources["operator.yaml"] =
		`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: camel-k-operator
  labels:
    app: "camel-k"
spec:
  replicas: 1
  selector:
    matchLabels:
      name: camel-k-operator
  template:
    metadata:
      labels:
        name: camel-k-operator
    spec:
      containers:
        - name: camel-k-operator
          image: docker.io/apache/camel-k:0.0.1-SNAPSHOT
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - camel-k-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "camel-k-operator"

`
	Resources["user-cluster-role.yaml"] =
		`
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: camel-k:edit
  labels:
    app: "camel-k"
    # Add these permissions to the "admin" and "edit" default roles.
    rbac.authorization.k8s.io/aggregate-to-admin: "true"
    rbac.authorization.k8s.io/aggregate-to-edit: "true"
rules:
- apiGroups: ["camel.apache.org"]
  resources: ["*"]
  verbs: ["*"]

`

}

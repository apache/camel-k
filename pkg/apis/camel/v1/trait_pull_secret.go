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

package v1

// The Pull Secret trait sets a pull secret on the pod,
// to allow Kubernetes to retrieve the container image from an external registry.
//
// The pull secret can be specified manually or, in case you've configured authentication for an external container registry
// on the `IntegrationPlatform`, the same secret is used to pull images.
//
// It's enabled by default whenever you configure authentication for an external container registry,
// so it assumes that external registries are private.
//
// If your registry does not need authentication for pulling images, you can disable this trait.
//
// +camel-k:trait=pull-secret.
type PullSecretTrait struct {
	Trait `property:",squash" json:",inline"`
	// The pull secret name to set on the Pod. If left empty this is automatically taken from the `IntegrationPlatform` registry configuration.
	SecretName string `property:"secret-name" json:"secretName,omitempty"`
	// When using a global operator with a shared platform, this enables delegation of the `system:image-puller` cluster role on the operator namespace to the integration service account.
	ImagePullerDelegation *bool `property:"image-puller-delegation" json:"imagePullerDelegation,omitempty"`
	// Automatically configures the platform registry secret on the pod if it is of type `kubernetes.io/dockerconfigjson`.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

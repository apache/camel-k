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

package trait

// The Pull Secret trait sets a pull secret on the pod to allow Kubernetes to retrieve
// the container image from an external registry.
//
// In a production environment it is highly advisable to provide such authentication
// and ensure the secret exists in the Integration namespace.
//
// +camel-k:trait=pull-secret.
//
//nolint:godoclint
type PullSecretTrait struct {
	Trait `json:",inline" property:",squash"`

	// The pull secret name to set on the Pod.
	SecretName string `json:"secretName,omitempty" property:"secret-name"`
	// Deprecated: no longer in use.
	DeprecatedImagePullerDelegation *bool `json:"imagePullerDelegation,omitempty" property:"image-puller-delegation"`
	// Deprecated: no longer in use.
	DeprecatedAuto *bool `json:"auto,omitempty" property:"auto"`
}

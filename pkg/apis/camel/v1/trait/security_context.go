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

import corev1 "k8s.io/api/core/v1"

// The Security Context trait can be used to configure the security setting of the Pod running the application.
//
// +camel-k:trait=security-context.
type SecurityContextTrait struct {
	PlatformBaseTrait `property:",squash" json:",inline"`

	// Security Context RunAsUser configuration (default none): this value is automatically retrieved in Openshift clusters when not explicitly set.
	RunAsUser *int64 `property:"run-as-user" json:"runAsUser,omitempty"`
	// Security Context RunAsNonRoot configuration (default false).
	RunAsNonRoot *bool `property:"run-as-non-root" json:"runAsNonRoot,omitempty"`
	// Security Context SeccompProfileType configuration (default RuntimeDefault).
	// +kubebuilder:validation:Enum=Unconfined;RuntimeDefault
	SeccompProfileType corev1.SeccompProfileType `property:"seccomp-profile-type" json:"seccompProfileType,omitempty"`
}

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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewIntegrationProfile --.
func NewIntegrationProfile(namespace string, name string) IntegrationProfile {
	return IntegrationProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationProfileKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// SetOperatorID sets the given operator id as an annotation.
func (in *IntegrationProfile) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

// GetTimeout returns the specified duration or a default one.
func (b *IntegrationProfileBuildSpec) GetTimeout() metav1.Duration {
	if b.Timeout == nil {
		return metav1.Duration{}
	}

	return *b.Timeout
}

var _ ResourceCondition = &IntegrationProfileCondition{}

// GetType --.
func (c *IntegrationProfileCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --.
func (c *IntegrationProfileCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --.
func (c *IntegrationProfileCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --.
func (c *IntegrationProfileCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --.
func (c *IntegrationProfileCondition) GetReason() string {
	return c.Reason
}

// GetMessage --.
func (c *IntegrationProfileCondition) GetMessage() string {
	return c.Message
}

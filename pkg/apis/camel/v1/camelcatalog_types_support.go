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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCamelCatalog --
func NewCamelCatalog(namespace string, name string) CamelCatalog {
	return CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewCamelCatalogWithSpecs --
func NewCamelCatalogWithSpecs(namespace string, name string, spec CamelCatalogSpec) CamelCatalog {
	return CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: spec,
	}
}

// NewCamelCatalogList --
func NewCamelCatalogList() CamelCatalogList {
	return CamelCatalogList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
	}
}

// GetType --
func (c CamelCatalogCondition) GetType() string {
	return string(c.Type)
}

// GetStatus --
func (c CamelCatalogCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// GetLastUpdateTime --
func (c CamelCatalogCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

// GetLastTransitionTime --
func (c CamelCatalogCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

// GetReason --
func (c CamelCatalogCondition) GetReason() string {
	return c.Reason
}

// GetMessage --
func (c CamelCatalogCondition) GetMessage() string {
	return c.Message
}

// GetConditions --
func (in *CamelCatalogStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

// GetCondition returns the condition with the provided type.
func (in *CamelCatalogStatus) GetCondition(condType CamelCatalogConditionType) *CamelCatalogCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition --
func (in *CamelCatalogStatus) SetCondition(condType CamelCatalogConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(CamelCatalogCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition --
func (in *CamelCatalogStatus) SetErrorCondition(condType CamelCatalogConditionType, reason string, err error) {
	in.SetConditions(CamelCatalogCondition{
		Type:               condType,
		Status:             corev1.ConditionFalse,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *CamelCatalogStatus) SetConditions(conditions ...CamelCatalogCondition) {
	for _, condition := range conditions {
		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = metav1.Now()
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.Now()
		}

		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
			return
		}
		// Do not update lastTransitionTime if the status of the condition doesn't change.
		if currentCond != nil && currentCond.Status == condition.Status {
			condition.LastTransitionTime = currentCond.LastTransitionTime
		}

		in.RemoveCondition(condition.Type)
		in.Conditions = append(in.Conditions, condition)
	}
}

// RemoveCondition removes the resource condition with the provided type.
func (in *CamelCatalogStatus) RemoveCondition(condType CamelCatalogConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

// GetRuntimeVersion returns the Camel K runtime version of the catalog.
func (c *CamelCatalogSpec) GetRuntimeVersion() string {
	return c.Runtime.Version
}

// GetCamelVersion returns the Camel version the runtime is based on.
func (c *CamelCatalogSpec) GetCamelVersion() string {
	return c.Runtime.Metadata["camel.version"]
}

// GetCamelQuarkusVersion returns the Camel Quarkus version the runtime is based on.
func (c *CamelCatalogSpec) GetCamelQuarkusVersion() string {
	return c.Runtime.Metadata["camel-quarkus.version"]
}

// GetQuarkusVersion returns the Quarkus version the runtime is based on.
func (c *CamelCatalogSpec) GetQuarkusVersion() string {
	return c.Runtime.Metadata["quarkus.version"]
}

// GetQuarkusToolingImage returns the Quarkus tooling image required to build an application based on this catalog.
func (c *CamelCatalogSpec) GetQuarkusToolingImage() string {
	return c.Runtime.Metadata["quarkus.native-builder-image"]
}

// HasCapability checks if the given capability is present in the catalog.
func (c *CamelCatalogSpec) HasCapability(capability string) bool {
	_, ok := c.Runtime.Capabilities[capability]

	return ok
}

// GetDependencyID returns a Camel K recognizable maven dependency for the artifact
func (in *CamelArtifact) GetDependencyID() string {
	switch {
	case in.GroupID == "org.apache.camel.quarkus" && strings.HasPrefix(in.ArtifactID, "camel-quarkus-"):
		return "camel:" + in.ArtifactID[14:]
	case in.Version == "":
		return "mvn:" + in.GroupID + ":" + in.ArtifactID
	default:
		return "mvn:" + in.GroupID + ":" + in.ArtifactID + ":" + in.Version
	}
}

func (in *CamelArtifact) GetConsumerDependencyIDs(schemeID string) []string {
	return in.getDependencyIDs(schemeID, consumerScheme)
}

func (in *CamelArtifact) GetProducerDependencyIDs(schemeID string) []string {
	return in.getDependencyIDs(schemeID, producerScheme)
}

func (in *CamelArtifact) getDependencyIDs(schemeID string, scope func(CamelScheme) CamelSchemeScope) []string {
	ads := in.getDependencies(schemeID, scope)
	if ads == nil {
		return nil
	}
	deps := make([]string, 0, len(ads))
	for _, ad := range ads {
		deps = append(deps, ad.GetDependencyID())
	}
	return deps
}

func (in *CamelArtifact) GetConsumerDependencies(schemeID string) []CamelArtifactDependency {
	return in.getDependencies(schemeID, consumerScheme)
}

func (in *CamelArtifact) GetProducerDependencies(schemeID string) []CamelArtifactDependency {
	return in.getDependencies(schemeID, producerScheme)
}

func (in *CamelArtifact) getDependencies(schemeID string, scope func(CamelScheme) CamelSchemeScope) []CamelArtifactDependency {
	scheme := in.GetScheme(schemeID)
	if scheme == nil {
		return nil
	}
	return scope(*scheme).Dependencies
}

func (in *CamelArtifact) GetScheme(schemeID string) *CamelScheme {
	for _, scheme := range in.Schemes {
		if scheme.ID == schemeID {
			return &scheme
		}
	}
	return nil
}

func consumerScheme(scheme CamelScheme) CamelSchemeScope {
	return scheme.Consumer
}

func producerScheme(scheme CamelScheme) CamelSchemeScope {
	return scheme.Producer
}

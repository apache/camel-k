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

// NewIntegration --
func NewIntegration(namespace string, name string) Integration {
	return Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewIntegrationList --
func NewIntegrationList() IntegrationList {
	return IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKind,
		},
	}
}

// Sources return a new slice containing all the sources associated to the integration
func (in *Integration) Sources() []SourceSpec {
	sources := make([]SourceSpec, 0, len(in.Spec.Sources)+len(in.Status.GeneratedSources))
	sources = append(sources, in.Spec.Sources...)
	sources = append(sources, in.Status.GeneratedSources...)

	return sources
}

// Resources return a new slice containing all the resources associated to the integration
func (in *Integration) Resources() []ResourceSpec {
	resources := make([]ResourceSpec, 0, len(in.Spec.Resources)+len(in.Status.GeneratedResources))
	resources = append(resources, in.Spec.Resources...)
	resources = append(resources, in.Status.GeneratedResources...)

	return resources
}

// AddSource --
func (in *IntegrationSpec) AddSource(name string, content string, language Language) {
	in.Sources = append(in.Sources, NewSourceSpec(name, content, language))
}

// AddSources --
func (in *IntegrationSpec) AddSources(sources ...SourceSpec) {
	in.Sources = append(in.Sources, sources...)
}

// AddResources --
func (in *IntegrationSpec) AddResources(resources ...ResourceSpec) {
	in.Resources = append(in.Resources, resources...)
}

// AddConfiguration --
func (in *IntegrationSpec) AddConfiguration(confType string, confValue string) {
	in.Configuration = append(in.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

// AddDependency --
func (in *IntegrationSpec) AddDependency(dependency string) {
	if in.Dependencies == nil {
		in.Dependencies = make([]string, 0)
	}
	newDep := dependency
	if strings.HasPrefix(newDep, "camel-quarkus") {
		newDep = "camel-quarkus:" + strings.TrimPrefix(dependency, "camel-quarkus-")
	} else if strings.HasPrefix(newDep, "camel-") {
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-")
	}
	for _, d := range in.Dependencies {
		if d == newDep {
			return
		}
	}
	in.Dependencies = append(in.Dependencies, newDep)
}

// Configurations --
func (in *IntegrationSpec) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

// Configurations --
func (in *IntegrationStatus) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

// Configurations --
func (in *Integration) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	answer := make([]ConfigurationSpec, 0)
	answer = append(answer, in.Status.Configuration...)
	answer = append(answer, in.Spec.Configuration...)

	return answer
}

// NewSourceSpec --
func NewSourceSpec(name string, content string, language Language) SourceSpec {
	return SourceSpec{
		DataSpec: DataSpec{
			Name:    name,
			Content: content,
		},
		Language: language,
	}
}

// NewResourceSpec --
func NewResourceSpec(name string, content string, destination string, resourceType ResourceType) ResourceSpec {
	return ResourceSpec{
		DataSpec: DataSpec{
			Name:    name,
			Content: content,
		},
		Type: resourceType,
	}
}

// InferLanguage returns the language of the source or discovers it from file extension if not set
func (in *SourceSpec) InferLanguage() Language {
	if in.Language != "" {
		return in.Language
	}
	for _, l := range Languages {
		if strings.HasSuffix(in.Name, "."+string(l)) {
			return l
		}
	}
	return ""
}

// SetIntegrationPlatform --
func (in *Integration) SetIntegrationPlatform(platform *IntegrationPlatform) {
	cs := corev1.ConditionTrue

	if platform.Status.Phase != IntegrationPlatformPhaseReady {
		cs = corev1.ConditionFalse
	}

	in.Status.SetCondition(IntegrationConditionPlatformAvailable, cs, IntegrationConditionPlatformAvailableReason, platform.Name)
	in.Status.Platform = platform.Name
}

// SetIntegrationKit --
func (in *Integration) SetIntegrationKit(kit *IntegrationKit) {
	cs := corev1.ConditionTrue

	if kit.Status.Phase != IntegrationKitPhaseReady {
		cs = corev1.ConditionFalse
	}

	in.Status.SetCondition(IntegrationConditionKitAvailable, cs, IntegrationConditionKitAvailableReason, kit.Name)
	in.Status.Kit = kit.Name
	in.Status.Image = kit.Status.Image
}

// GetCondition returns the condition with the provided type.
func (in *IntegrationStatus) GetCondition(condType IntegrationConditionType) *IntegrationCondition {
	for i := range in.Conditions {
		c := in.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition --
func (in *IntegrationStatus) SetCondition(condType IntegrationConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(IntegrationCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetErrorCondition --
func (in *IntegrationStatus) SetErrorCondition(condType IntegrationConditionType, reason string, err error) {
	in.SetConditions(IntegrationCondition{
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
func (in *IntegrationStatus) SetConditions(conditions ...IntegrationCondition) {
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
func (in *IntegrationStatus) RemoveCondition(condType IntegrationConditionType) {
	newConditions := in.Conditions[:0]
	for _, c := range in.Conditions {
		if c.Type != condType {
			newConditions = append(newConditions, c)
		}
	}

	in.Conditions = newConditions
}

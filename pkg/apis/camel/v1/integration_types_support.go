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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const IntegrationLabel = "camel.apache.org/integration"

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

func NewIntegrationList() IntegrationList {
	return IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKind,
		},
	}
}

func (in *Integration) Initialize() {
	profile := in.Status.Profile
	if in.Spec.Profile != "" {
		profile = in.Spec.Profile
	}
	in.Status = IntegrationStatus{
		Phase:   IntegrationPhaseInitialization,
		Profile: profile,
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

func (in *IntegrationSpec) AddSource(name string, content string, language Language) {
	in.Sources = append(in.Sources, NewSourceSpec(name, content, language))
}

func (in *IntegrationSpec) AddSources(sources ...SourceSpec) {
	in.Sources = append(in.Sources, sources...)
}

func (in *IntegrationSpec) AddResources(resources ...ResourceSpec) {
	in.Resources = append(in.Resources, resources...)
}

func (in *IntegrationSpec) AddFlows(flows ...Flow) {
	in.Flows = append(in.Flows, flows...)
}

func (in *IntegrationSpec) AddConfiguration(confType string, confValue string) {
	in.Configuration = append(in.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

func (in *IntegrationSpec) AddDependency(dependency string) {
	if in.Dependencies == nil {
		in.Dependencies = make([]string, 0)
	}
	newDep := NormalizeDependency(dependency)
	for _, d := range in.Dependencies {
		if d == newDep {
			return
		}
	}
	in.Dependencies = append(in.Dependencies, newDep)
}

// NormalizeDependency converts different forms of camel dependencies
// -- `camel-xxx`, `camel-quarkus-xxx`, and `camel-quarkus:xxx` --
// into the unified form `camel:xxx`.
func NormalizeDependency(dependency string) string {
	newDep := dependency
	switch {
	case strings.HasPrefix(newDep, "camel-quarkus-"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-quarkus-")
	case strings.HasPrefix(newDep, "camel-quarkus:"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-quarkus:")
	case strings.HasPrefix(newDep, "camel-"):
		newDep = "camel:" + strings.TrimPrefix(dependency, "camel-")
	}
	return newDep
}

// GetConfigurationProperty returns a configuration property
func (in *IntegrationSpec) GetConfigurationProperty(property string) string {
	for _, confSpec := range in.Configuration {
		if confSpec.Type == "property" && strings.HasPrefix(confSpec.Value, property) {
			splitConf := strings.Split(confSpec.Value, "=")
			if len(splitConf) > 1 {
				return trimFirstLeadingSpace(splitConf[1])
			}
		}
	}
	return ""
}

func trimFirstLeadingSpace(val string) string {
	if strings.HasPrefix(val, " ") {
		return val[1:]
	}
	return val
}

func (in *IntegrationStatus) AddOrReplaceGeneratedResources(resources ...ResourceSpec) {
	newResources := make([]ResourceSpec, 0)
	for _, resource := range resources {
		replaced := false
		for i, r := range in.GeneratedResources {
			if r.Name == resource.Name {
				in.GeneratedResources[i] = resource
				replaced = true
				break
			}
		}
		if !replaced {
			newResources = append(newResources, resource)
		}
	}

	in.GeneratedResources = append(in.GeneratedResources, newResources...)
}

func (in *IntegrationStatus) AddOrReplaceGeneratedSources(sources ...SourceSpec) {
	newSources := make([]SourceSpec, 0)
	for _, source := range sources {
		replaced := false
		for i, r := range in.GeneratedSources {
			if r.Name == source.Name {
				in.GeneratedSources[i] = source
				replaced = true
				break
			}
		}
		if !replaced {
			newSources = append(newSources, source)
		}
	}

	in.GeneratedSources = append(in.GeneratedSources, newSources...)
}

func (in *IntegrationStatus) AddConfigurationsIfMissing(configurations ...ConfigurationSpec) {
	for _, config := range configurations {
		alreadyPresent := false
		for _, r := range in.Configuration {
			if r.Type == config.Type && r.Value == config.Value {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			in.Configuration = append(in.Configuration, config)
		}
	}
}

func (in *IntegrationSpec) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

func (in *IntegrationStatus) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	return in.Configuration
}

func (in *Integration) Configurations() []ConfigurationSpec {
	if in == nil {
		return []ConfigurationSpec{}
	}

	answer := make([]ConfigurationSpec, 0)
	answer = append(answer, in.Status.Configuration...)
	answer = append(answer, in.Spec.Configuration...)

	return answer
}

func NewSourceSpec(name string, content string, language Language) SourceSpec {
	return SourceSpec{
		DataSpec: DataSpec{
			Name:    name,
			Content: content,
		},
		Language: language,
	}
}

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

// SetOperatorID sets the given operator id as an annotation
func (in *Integration) SetOperatorID(operatorID string) {
	SetAnnotation(&in.ObjectMeta, OperatorIDAnnotation, operatorID)
}

func (in *Integration) SetIntegrationPlatform(platform *IntegrationPlatform) {
	cs := corev1.ConditionTrue

	if platform.Status.Phase != IntegrationPlatformPhaseReady {
		cs = corev1.ConditionFalse
	}

	in.Status.SetCondition(IntegrationConditionPlatformAvailable, cs, IntegrationConditionPlatformAvailableReason, platform.Namespace+"/"+platform.Name)
	in.Status.Platform = platform.Name
}

func (in *Integration) SetIntegrationKit(kit *IntegrationKit) {
	if kit == nil {
		in.Status.IntegrationKit = nil
		return
	}

	cs := corev1.ConditionTrue
	message := kit.Name
	if kit.Status.Phase != IntegrationKitPhaseReady {
		cs = corev1.ConditionFalse
		if kit.Status.Phase == IntegrationKitPhaseNone {
			message = fmt.Sprintf("creating a new integration kit")
		} else {
			message = fmt.Sprintf("integration kit %s/%s is in state %q", kit.Namespace, kit.Name, kit.Status.Phase)
		}
	}

	in.Status.SetCondition(IntegrationConditionKitAvailable, cs, IntegrationConditionKitAvailableReason, message)
	in.Status.IntegrationKit = &corev1.ObjectReference{
		Namespace: kit.Namespace,
		Name:      kit.Name,
	}
	image := kit.Status.Image
	if image == "" {
		image = kit.Spec.Image
	}
	in.Status.Image = image
}

func (in *Integration) GetIntegrationKitNamespace(p *IntegrationPlatform) string {
	if in.Status.IntegrationKit != nil && in.Status.IntegrationKit.Namespace != "" {
		return in.Status.IntegrationKit.Namespace
	}
	if in.Spec.IntegrationKit != nil && in.Spec.IntegrationKit.Namespace != "" {
		return in.Spec.IntegrationKit.Namespace
	}
	if p != nil {
		return p.Namespace
	}
	return in.Namespace
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

func (in *IntegrationStatus) SetCondition(condType IntegrationConditionType, status corev1.ConditionStatus, reason string, message string) {
	in.SetConditions(IntegrationCondition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func (in *IntegrationStatus) SetErrorCondition(condType IntegrationConditionType, reason string, err error) {
	in.SetConditions(IntegrationCondition{
		Type:    condType,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: err.Error(),
	})
}

// SetConditions updates the resource to include the provided conditions.
//
// If a condition that we are about to add already exists and has the same status and
// reason then we are not going to update.
func (in *IntegrationStatus) SetConditions(conditions ...IntegrationCondition) {
	// Round to second precision, as meta.Time fields are marshalled in RFC3339 format
	now := metav1.Now().Rfc3339Copy()
	for _, condition := range conditions {
		currentCond := in.GetCondition(condition.Type)

		if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
			break
		}

		if condition.LastUpdateTime.IsZero() {
			condition.LastUpdateTime = now
		}

		if condition.LastTransitionTime.IsZero() {
			// We may want not to set it when the current condition is nil
			condition.LastTransitionTime = now
		}

		if (condition.FirstTruthyTime == nil || condition.FirstTruthyTime.IsZero()) && condition.Status == corev1.ConditionTrue {
			condition.FirstTruthyTime = &now
		}

		if currentCond != nil {
			if currentCond.Status == condition.Status {
				// Do not update LastTransitionTime if the status of the condition doesn't change
				condition.LastTransitionTime = currentCond.LastTransitionTime
			}
			if currentCond.FirstTruthyTime != nil && !currentCond.FirstTruthyTime.IsZero() {
				// Preserve FirstTruthyTime
				condition.FirstTruthyTime = currentCond.FirstTruthyTime.DeepCopy()
			}
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

var _ ResourceCondition = IntegrationCondition{}

func (in *IntegrationStatus) GetConditions() []ResourceCondition {
	res := make([]ResourceCondition, 0, len(in.Conditions))
	for _, c := range in.Conditions {
		res = append(res, c)
	}
	return res
}

func (c IntegrationCondition) GetType() string {
	return string(c.Type)
}

func (c IntegrationCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

func (c IntegrationCondition) GetLastUpdateTime() metav1.Time {
	return c.LastUpdateTime
}

func (c IntegrationCondition) GetLastTransitionTime() metav1.Time {
	return c.LastTransitionTime
}

func (c IntegrationCondition) GetReason() string {
	return c.Reason
}

func (c IntegrationCondition) GetMessage() string {
	return c.Message
}

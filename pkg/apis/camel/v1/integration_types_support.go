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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	yaml2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// IntegrationLabel is used to tag k8s object created by a given Integration.
	IntegrationLabel = "camel.apache.org/integration"
	// IntegrationGenerationLabel is used to check on outdated integration resources that can be removed by garbage collection.
	IntegrationGenerationLabel = "camel.apache.org/generation"
	// IntegrationSyntheticLabel is used to tag k8s synthetic Integrations.
	IntegrationSyntheticLabel = "camel.apache.org/is-synthetic"
	// IntegrationImportedKindLabel specifies from what kind of resource an Integration was imported.
	IntegrationImportedKindLabel = "camel.apache.org/imported-from-kind"
	// IntegrationImportedNameLabel specifies from what resource an Integration was imported.
	IntegrationImportedNameLabel = "camel.apache.org/imported-from-name"

	// IntegrationFlowEmbeddedSourceName --.
	IntegrationFlowEmbeddedSourceName = "camel-k-embedded-flow.yaml"
)

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

// AllSources returns a new slice containing all the sources associated to the Integration.
// It merges any generated source, giving priority to this if the same
// source exist both in spec and status.
func (in *Integration) AllSources() []SourceSpec {
	var sources []SourceSpec
	sources = append(sources, in.Status.GeneratedSources...)
	for _, src := range in.Spec.Sources {
		if len(in.Status.GeneratedSources) == 0 {
			sources = append(sources, src)
		} else {
			for _, genSrc := range in.Status.GeneratedSources {
				if src.Name != genSrc.Name {
					sources = append(sources, src)
				}
			}
		}
	}

	return sources
}

// OriginalSources return a new slice containing only the original sources provided within the Integration.
// It checks if the spec source was transformed and available in the status, and return the latter in such a case.
func (in *Integration) OriginalSources() []SourceSpec {
	var sources []SourceSpec
	for _, src := range in.Spec.Sources {
		found := false
	loop:
		for _, genSrc := range in.Status.GeneratedSources {
			if src.Name == genSrc.Name {
				sources = append(sources, genSrc)
				found = true
				break loop
			}
		}
		if !found {
			sources = append(sources, src)
		}
	}

	return sources
}

// OriginalSourcesOnly return a new slice containing only the original sources provided within the Integration spec
// including the embedded yaml flow if it exists.
func (in *Integration) OriginalSourcesOnly() []SourceSpec {
	var sources []SourceSpec
	sources = append(sources, in.Spec.Sources...)
	if len(in.Spec.Flows) > 0 {
		content, _ := ToYamlDSL(in.Spec.Flows)
		sources = append(sources, SourceSpec{
			DataSpec: DataSpec{
				Name:    IntegrationFlowEmbeddedSourceName,
				Content: string(content),
			},
		})
	}

	return sources
}

// IsManagedBuild returns true when the Integration requires to be built by the operator.
func (in *Integration) IsManagedBuild() bool {
	if in.Spec.Traits.Container == nil || in.Spec.Traits.Container.Image == "" {
		return true
	}
	isManagedBuild, err := regexp.MatchString("(.*)/(.*)/camel-k-kit-(.*)@sha256:(.*)", in.Spec.Traits.Container.Image)
	return err == nil && isManagedBuild
}

func (in *IntegrationSpec) AddSource(name string, content string, language Language) {
	in.Sources = append(in.Sources, NewSourceSpec(name, content, language))
}

func (in *IntegrationSpec) AddSources(sources ...SourceSpec) {
	in.Sources = append(in.Sources, sources...)
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
	for _, d := range in.Dependencies {
		if d == dependency {
			return
		}
	}
	in.Dependencies = append(in.Dependencies, dependency)
}

// AddConfigurationProperty adds a new configuration property.
func (in *IntegrationSpec) AddConfigurationProperty(confValue string) {
	in.AddConfiguration("property", confValue)
}

// GetConfigurationProperty returns a configuration property.
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

// SetOperatorID sets the given operator id as an annotation.
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
			message = "creating a new integration kit"
		} else {
			message = fmt.Sprintf("integration kit %s/%s is in state %q", kit.Namespace, kit.Name, kit.Status.Phase)
			if kit.Status.Phase == IntegrationKitPhaseError && kit.Status.Failure != nil {
				// Append specific reason for the failure
				message = message + ". Failure: " + kit.Status.Failure.Reason
			}
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

// IsConditionTrue checks if the condition with the given type is true.
func (in *Integration) IsConditionTrue(conditionType IntegrationConditionType) bool {
	if in == nil {
		return false
	}
	cond := in.Status.GetCondition(conditionType)
	if cond == nil {
		return false
	}

	return cond.Status == corev1.ConditionTrue
}

// SetReadyCondition sets Ready condition with the given status, reason, and message.
func (in *Integration) SetReadyCondition(status corev1.ConditionStatus, reason, message string) {
	in.Status.SetCondition(IntegrationConditionReady, status, reason, message)
}

// SetReadyConditionError sets Ready condition to False with the given error message.
func (in *Integration) SetReadyConditionError(err string) {
	in.SetReadyCondition(corev1.ConditionFalse, IntegrationConditionErrorReason, err)
}

// IsSynthetic returns true for synthetic Integrations (non managed, likely imported from external deployments).
func (in *Integration) IsSynthetic() bool {
	return in.Annotations[IntegrationSyntheticLabel] == "true"
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

// FromYamlDSLString creates a slice of flows from a Camel YAML DSL string.
func FromYamlDSLString(flowsString string) ([]Flow, error) {
	return FromYamlDSL(bytes.NewReader([]byte(flowsString)))
}

// FromYamlDSL creates a slice of flows from a Camel YAML DSL stream.
func FromYamlDSL(reader io.Reader) ([]Flow, error) {
	buffered, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var flows []Flow
	// Using the Kubernetes decoder to turn them into JSON before unmarshal.
	// This avoids having map[interface{}]interface{} objects which are not JSON compatible.
	jsonData, err := yaml.ToJSON(buffered)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(jsonData, &flows); err != nil {
		return nil, err
	}
	return flows, err
}

// ToYamlDSL converts a flow into its Camel YAML DSL equivalent.
func ToYamlDSL(flows []Flow) ([]byte, error) {
	data, err := json.Marshal(&flows)
	if err != nil {
		return nil, err
	}
	var jsondata interface{}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(&jsondata); err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}
	yamldata, err := yaml2.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %w", err)
	}

	return yamldata, nil
}

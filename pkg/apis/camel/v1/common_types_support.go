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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/imdario/mergo"
)

func (in *Artifact) String() string {
	return in.ID
}

func (in *ConfigurationSpec) String() string {
	return fmt.Sprintf("%s=%s", in.Type, in.Value)
}

func (in *RuntimeSpec) CapabilityDependencies(capability string) []MavenArtifact {
	deps := make([]MavenArtifact, 0)

	if capability, ok := in.Capabilities[capability]; ok {
		deps = append(deps, capability.Dependencies...)
	}

	return deps
}

// TraitProfileByName returns the trait profile corresponding to the given name (case insensitive).
func TraitProfileByName(name string) TraitProfile {
	for _, p := range AllTraitProfiles {
		if strings.EqualFold(name, string(p)) {
			return p
		}
	}
	return ""
}

// Equal checks if the profile is equal to the given profile (case insensitive).
func (p TraitProfile) Equal(other TraitProfile) bool {
	return strings.EqualFold(string(p), string(other))
}

// Merge merges the given Traits into the receiver.
func (t *Traits) Merge(other Traits) error {
	// marshal both
	data1, err := json.Marshal(t)
	if err != nil {
		return err
	}
	data2, err := json.Marshal(other)
	if err != nil {
		return err
	}

	// merge them
	map1 := make(map[string]interface{})
	if err := json.Unmarshal(data1, &map1); err != nil {
		return err
	}
	map2 := make(map[string]interface{})
	if err := json.Unmarshal(data2, &map2); err != nil {
		return err
	}
	// values from merged trait take precedence over the original ones
	if err := mergo.Merge(&map1, map2, mergo.WithOverride); err != nil {
		return err
	}

	// unmarshal it
	data, err := json.Marshal(map1)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &t); err != nil {
		return err
	}

	return nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// String returns a string representation of RawMessage.
func (m *RawMessage) String() string {
	if m == nil {
		return ""
	}
	b, err := m.MarshalJSON()
	if err != nil {
		return ""
	}
	return string(b)
}

var _ json.Marshaler = (*RawMessage)(nil)
var _ json.Unmarshaler = (*RawMessage)(nil)

// GetOperatorIDAnnotation to safely get the operator id annotation value.
func GetOperatorIDAnnotation(obj metav1.Object) string {
	if obj == nil || obj.GetAnnotations() == nil {
		return ""
	}

	if operatorID, ok := obj.GetAnnotations()[OperatorIDAnnotation]; ok {
		return operatorID
	}

	return ""
}

// SetAnnotation safely sets the annotation on the given runtime object.
func SetAnnotation(obj *metav1.ObjectMeta, name string, value string) {
	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}

	obj.Annotations[name] = value
}

var _ json.Marshaler = (*RawMessage)(nil)
var _ json.Unmarshaler = (*RawMessage)(nil)

// IsEmpty -- .
func (bc *BuildConfiguration) IsEmpty() bool {
	return bc.Strategy == "" &&
		bc.OrderStrategy == "" &&
		bc.RequestCPU == "" &&
		bc.RequestMemory == "" &&
		bc.LimitCPU == "" &&
		bc.LimitMemory == ""
}

// DecodeValueSource returns a ValueSource object from an input that respects the format configmap|secret:resource-name[/path].
func DecodeValueSource(input string, defaultKey string, errorMessage string) (ValueSource, error) {
	sub := make([]string, 0)
	rex := regexp.MustCompile(`^(configmap|secret):([a-zA-Z0-9][a-zA-Z0-9-]*)(/([a-zA-Z0-9].*))?$`)
	hits := rex.FindAllStringSubmatch(input, -1)

	for _, hit := range hits {
		if len(hit) > 1 {
			sub = append(sub, hit[1:]...)
		}
	}

	// nolint: gosec // sub[3] and sub[0] cannot be out of bounds
	if len(sub) >= 2 {
		key := defaultKey
		if len(sub) == 4 && sub[3] != "" {
			key = sub[3]
		}

		if sub[0] == "configmap" {
			return ValueSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
		if sub[0] == "secret" {
			return ValueSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sub[1],
					},
					Key: key,
				},
			}, nil
		}
	}

	return ValueSource{}, fmt.Errorf(errorMessage)
}

// IsGeneratedFromKamelet determines is a source spec is derived from a Kamelet.
func (s *SourceSpec) IsGeneratedFromKamelet() bool {
	return s.FromKamelet
}

// InferLanguage returns the language of the source or discovers it from file extension if not set.
func (s *SourceSpec) InferLanguage() Language {
	if s.Language != "" {
		return s.Language
	}
	for _, l := range Languages {
		if strings.HasSuffix(s.Name, "."+string(l)) {
			return l
		}
	}
	return ""
}

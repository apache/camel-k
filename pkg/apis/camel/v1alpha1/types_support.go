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

package v1alpha1

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// **********************************
//
// Methods
//
// **********************************

func (spec ConfigurationSpec) String() string {
	return fmt.Sprintf("%s=%s", spec.Type, spec.Value)
}

// **********************************
//
// Helpers
//
// **********************************

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
func NewResourceSpec(name string, content string, destination string) ResourceSpec {
	return ResourceSpec{
		DataSpec: DataSpec{
			Name:    name,
			Content: content,
		},
	}
}

// NewIntegrationPlatformList --
func NewIntegrationPlatformList() IntegrationPlatformList {
	return IntegrationPlatformList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationPlatformKind,
		},
	}
}

// NewIntegrationPlatform --
func NewIntegrationPlatform(namespace string, name string) IntegrationPlatform {
	return IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationPlatformKind,
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

// NewIntegrationContext --
func NewIntegrationContext(namespace string, name string) IntegrationContext {
	return IntegrationContext{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationContextKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewIntegrationContextList --
func NewIntegrationContextList() IntegrationContextList {
	return IntegrationContextList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationContextKind,
		},
	}
}

// TraitProfileByName returns the trait profile corresponding to the given name (case insensitive)
func TraitProfileByName(name string) TraitProfile {
	for _, p := range allTraitProfiles {
		if strings.EqualFold(name, string(p)) {
			return p
		}
	}
	return ""
}

// Serialize serializes a Flow
func (flows Flows) Serialize() (string, error) {
	res, err := yaml.Marshal(flows)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// InferLanguage returns the language of the source or discovers it from file extension if not set
func (s SourceSpec) InferLanguage() Language {
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

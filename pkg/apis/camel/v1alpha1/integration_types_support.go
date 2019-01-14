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
	"strings"

	"github.com/apache/camel-k/pkg/util"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewIntegrationList --
func NewIntegrationList() IntegrationList {
	return IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationKind,
		},
	}
}

// Sources ---
func (i *Integration) Sources() []SourceSpec {
	allSources := make([]SourceSpec, 0, len(i.Spec.Sources)+len(i.Status.GeneratedSources))
	allSources = append(allSources, i.Spec.Sources...)
	allSources = append(allSources, i.Status.GeneratedSources...)

	return allSources
}

// AddSource --
func (is *IntegrationSpec) AddSource(name string, content string, language Language) {
	is.Sources = append(is.Sources, NewSourceSpec(name, content, language))
}

// AddSources --
func (is *IntegrationSpec) AddSources(sources ...SourceSpec) {
	is.Sources = append(is.Sources, sources...)
}

// AddResources --
func (is *IntegrationSpec) AddResources(resources ...ResourceSpec) {
	is.Resources = append(is.Resources, resources...)
}

// AddConfiguration --
func (is *IntegrationSpec) AddConfiguration(confType string, confValue string) {
	is.Configuration = append(is.Configuration, ConfigurationSpec{
		Type:  confType,
		Value: confValue,
	})
}

// AddDependency --
func (is *IntegrationSpec) AddDependency(dependency string) {
	switch {
	case strings.HasPrefix(dependency, "mvn:"):
		util.StringSliceUniqueAdd(&is.Dependencies, dependency)
	case strings.HasPrefix(dependency, "file:"):
		util.StringSliceUniqueAdd(&is.Dependencies, dependency)
	case strings.HasPrefix(dependency, "camel-"):
		util.StringSliceUniqueAdd(&is.Dependencies, "camel:"+strings.TrimPrefix(dependency, "camel-"))
	}
}

// Decode the trait configuration to a type safe struct
func (in *IntegrationTraitSpec) Decode(target interface{}) error {
	md := mapstructure.Metadata{}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &target,
		},
	)

	if err != nil {
		return err
	}

	return decoder.Decode(in.Configuration)
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

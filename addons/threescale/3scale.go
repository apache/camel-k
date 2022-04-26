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

package threescale

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/trait"
)

// The 3scale trait can be used to automatically create annotations that allow
// 3scale to discover the generated service and make it available for API management.
//
// The 3scale trait is disabled by default.
//
// +camel-k:trait=3scale.
type Trait struct {
	traitv1.Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The scheme to use to contact the service (default `http`)
	Scheme string `property:"scheme" json:"scheme,omitempty"`
	// The path where the API is published (default `/`)
	Path string `property:"path" json:"path,omitempty"`
	// The port where the service is exposed (default `80`)
	Port int `property:"port" json:"port,omitempty"`
	// The path where the Open-API specification is published (default `/openapi.json`)
	DescriptionPath *string `property:"description-path" json:"descriptionPath,omitempty"`
}

type threeScaleTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

const (
	// ThreeScaleSchemeAnnotation --.
	ThreeScaleSchemeAnnotation = "discovery.3scale.net/scheme"
	// ThreeScaleSchemeDefaultValue --.
	ThreeScaleSchemeDefaultValue = "http"

	// ThreeScalePortAnnotation --.
	ThreeScalePortAnnotation = "discovery.3scale.net/port"
	// ThreeScalePortDefaultValue --.
	ThreeScalePortDefaultValue = 80

	// ThreeScalePathAnnotation --.
	ThreeScalePathAnnotation = "discovery.3scale.net/path"
	// ThreeScalePathDefaultValue --.
	ThreeScalePathDefaultValue = "/"

	// ThreeScaleDescriptionPathAnnotation --.
	ThreeScaleDescriptionPathAnnotation = "discovery.3scale.net/description-path"
	// ThreeScaleDescriptionPathDefaultValue --.
	ThreeScaleDescriptionPathDefaultValue = "/openapi.json"

	// ThreeScaleDiscoveryLabel --.
	ThreeScaleDiscoveryLabel = "discovery.3scale.net"
	// ThreeScaleDiscoveryLabelEnabled --.
	ThreeScaleDiscoveryLabelEnabled = "true"
)

// NewThreeScaleTrait --.
func NewThreeScaleTrait() trait.Trait {
	return &threeScaleTrait{
		BaseTrait: trait.NewBaseTrait("3scale", trait.TraitOrderPostProcessResources),
	}
}

func (t *threeScaleTrait) Configure(e *trait.Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		// disabled by default
		return false, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.Scheme == "" {
			t.Scheme = ThreeScaleSchemeDefaultValue
		}
		if t.Path == "" {
			t.Path = ThreeScalePathDefaultValue
		}
		if t.Port == 0 {
			t.Port = ThreeScalePortDefaultValue
		}
		if t.DescriptionPath == nil {
			openAPI := ThreeScaleDescriptionPathDefaultValue
			t.DescriptionPath = &openAPI
		}
	}
	return true, nil
}

func (t *threeScaleTrait) Apply(e *trait.Environment) error {
	if svc := e.Resources.GetServiceForIntegration(e.Integration); svc != nil {
		t.addLabelsAndAnnotations(&svc.ObjectMeta)
	}
	return nil
}

func (t *threeScaleTrait) addLabelsAndAnnotations(obj *metav1.ObjectMeta) {
	if obj.Labels == nil {
		obj.Labels = make(map[string]string)
	}
	obj.Labels[ThreeScaleDiscoveryLabel] = ThreeScaleDiscoveryLabelEnabled

	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	if t.Scheme != "" {
		obj.Annotations[ThreeScaleSchemeAnnotation] = t.Scheme
	}
	if t.Path != "" {
		obj.Annotations[ThreeScalePathAnnotation] = t.Path
	}
	if t.Port != 0 {
		obj.Annotations[ThreeScalePortAnnotation] = strconv.Itoa(t.Port)
	}
	if t.DescriptionPath != nil && *t.DescriptionPath != "" {
		obj.Annotations[ThreeScaleDescriptionPathAnnotation] = *t.DescriptionPath
	}
}

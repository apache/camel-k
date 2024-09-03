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

import (
	"strconv"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type threeScaleTrait struct {
	BaseTrait
	traitv1.ThreeScaleTrait `property:",squash"`
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
func NewThreeScaleTrait() Trait {
	return &threeScaleTrait{
		BaseTrait: NewBaseTrait("3scale", TraitOrderPostProcessResources),
	}
}

func (t *threeScaleTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}
	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if ptr.Deref(t.Auto, true) {
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

	return true, nil, nil
}

func (t *threeScaleTrait) Apply(e *Environment) error {
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

	if t.Scheme != "" {
		v1.SetAnnotation(obj, ThreeScaleSchemeAnnotation, t.Scheme)
	}
	if t.Path != "" {
		v1.SetAnnotation(obj, ThreeScalePathAnnotation, t.Path)
	}
	if t.Port != 0 {
		v1.SetAnnotation(obj, ThreeScalePortAnnotation, strconv.Itoa(t.Port))
	}
	if t.DescriptionPath != nil && *t.DescriptionPath != "" {
		v1.SetAnnotation(obj, ThreeScaleDescriptionPathAnnotation, *t.DescriptionPath)
	}
}

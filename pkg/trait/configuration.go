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
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/property"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// The configuration trait is used to customize the Integration configuration such as properties and resources.
//
// +camel-k:trait=configuration
type configurationTrait struct {
	BaseTrait `property:",squash"`
	// A list of properties to be provided to the Integration runtime
	Properties []string `property:"properties" json:"properties,omitempty"`
}

func newConfigurationTrait() Trait {
	return &configurationTrait{
		BaseTrait: NewBaseTrait("configuration", 700),
	}
}

func (t *configurationTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return true, nil
}

func (t *configurationTrait) Apply(e *Environment) error {
	if e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
		e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning) {
		// Get all resources
		maps := e.computeConfigMaps()
		if t.Properties != nil {
			// Only user.properties
			maps = append(maps, t.computeUserProperties(e)...)
		}
		e.Resources.AddAll(maps)
	}

	return nil
}

func (t *configurationTrait) IsPlatformTrait() bool {
	return true
}

func (t *configurationTrait) computeUserProperties(e *Environment) []ctrl.Object {
	maps := make([]ctrl.Object, 0)

	// combine properties of integration with kit, integration
	// properties have the priority
	userProperties := ""

	for _, prop := range t.Properties {
		k, v := property.SplitPropertyFileEntry(prop)
		userProperties += fmt.Sprintf("%s=%s\n", k, v)
	}

	if userProperties != "" {
		maps = append(
			maps,
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      e.Integration.Name + "-user-properties",
					Namespace: e.Integration.Namespace,
					Labels: map[string]string{
						v1.IntegrationLabel:                e.Integration.Name,
						"camel.apache.org/properties.type": "user",
					},
				},
				Data: map[string]string{
					"application.properties": userProperties,
				},
			},
		)
	}

	return maps
}

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
	"strings"

	"gopkg.in/yaml.v2"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
)

// The error-handler is a platform trait used to inject Error Handler source into the integration runtime.
//
// +camel-k:trait=error-handler.
type errorHandlerTrait struct {
	BaseTrait `property:",squash"`
	// The error handler ref name provided or found in application properties
	ErrorHandlerRef string `property:"ref" json:"ref,omitempty"`
}

func newErrorHandlerTrait() Trait {
	return &errorHandlerTrait{
		// NOTE: Must run before dependency trait
		BaseTrait: NewBaseTrait("error-handler", 470),
	}
}

// IsPlatformTrait overrides base class method.
func (t *errorHandlerTrait) IsPlatformTrait() bool {
	return true
}

func (t *errorHandlerTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if t.ErrorHandlerRef == "" {
		t.ErrorHandlerRef = e.Integration.Spec.GetConfigurationProperty(v1alpha1.ErrorHandlerRefName)
	}

	return t.ErrorHandlerRef != "", nil
}

func (t *errorHandlerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// If the user configure directly the URI, we need to auto-discover the underlying component
		// and add the related dependency
		defaultErrorHandlerURI := e.Integration.Spec.GetConfigurationProperty(
			fmt.Sprintf("%s.deadLetterUri", v1alpha1.ErrorHandlerAppPropertiesPrefix))
		if defaultErrorHandlerURI != "" && !strings.HasPrefix(defaultErrorHandlerURI, "kamelet:") {
			t.addErrorHandlerDependencies(e, defaultErrorHandlerURI)
		}

		return t.addErrorHandlerAsSource(e)
	}
	return nil
}

func (t *errorHandlerTrait) addErrorHandlerDependencies(e *Environment, uri string) {
	candidateComp, scheme := e.CamelCatalog.DecodeComponent(uri)
	if candidateComp != nil {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, candidateComp.GetDependencyID())
		if scheme != nil {
			for _, dep := range candidateComp.GetProducerDependencyIDs(scheme.ID) {
				util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dep)
			}
		}
	}
}

func (t *errorHandlerTrait) addErrorHandlerAsSource(e *Environment) error {
	flowErrorHandler := map[string]interface{}{
		"error-handler": map[string]string{
			"ref": t.ErrorHandlerRef,
		},
	}
	encodedFlowErrorHandler, err := yaml.Marshal([]map[string]interface{}{flowErrorHandler})
	if err != nil {
		return err
	}
	errorHandlerSource := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "camel-k-embedded-error-handler.yaml",
			Content: string(encodedFlowErrorHandler),
		},
		Language: v1.LanguageYaml,
		Type:     v1.SourceTypeErrorHandler,
	}

	e.Integration.Status.AddOrReplaceGeneratedSources(errorHandlerSource)

	return nil
}

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
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/property"

	"github.com/apache/camel-k/v2/pkg/util"
)

type errorHandlerTrait struct {
	BasePlatformTrait
	traitv1.ErrorHandlerTrait `property:",squash"`
}

func newErrorHandlerTrait() Trait {
	return &errorHandlerTrait{
		// NOTE: Must run before dependency trait
		BasePlatformTrait: NewBasePlatformTrait("error-handler", 470),
	}
}

func (t *errorHandlerTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if t.ErrorHandlerRef == "" {
		t.ErrorHandlerRef = e.Integration.Spec.GetConfigurationProperty(v1.ErrorHandlerRefName)
	}

	return t.ErrorHandlerRef != "", nil, nil
}

func (t *errorHandlerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// If the user configure directly the URI, we need to auto-discover the underlying component
		// and add the related dependency
		defaultErrorHandlerURI := e.Integration.Spec.GetConfigurationProperty(
			fmt.Sprintf("%s.deadLetterUri", v1.ErrorHandlerAppPropertiesPrefix))
		if defaultErrorHandlerURI != "" && !strings.HasPrefix(defaultErrorHandlerURI, "kamelet:") {
			t.addErrorHandlerDependencies(e, defaultErrorHandlerURI)
		}

		if shouldHandleNoErrorHandler(e.Integration) {
			// noErrorHandler is enabled by default on Kamelets since Camel 4.4.0 (runtimeVersion 3.8.0)
			// need to disable this setting so that pipe error handler works
			confValue, err := property.EncodePropertyFileEntry("camel.component.kamelet.noErrorHandler", "false")
			if err != nil {
				return err
			}

			e.Integration.Spec.AddConfigurationProperty(confValue)
		}

		return t.addGlobalErrorHandlerAsSource(e)
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

func (t *errorHandlerTrait) addGlobalErrorHandlerAsSource(e *Environment) error {
	flowErrorHandler := map[string]interface{}{
		"error-handler": map[string]string{
			"ref-error-handler": t.ErrorHandlerRef,
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

// shouldHandleNoErrorHandler determines the runtime version and checks on noErrorHandler that is configured for this version.
func shouldHandleNoErrorHandler(it *v1.Integration) bool {
	if it.Status.RuntimeVersion != "" {
		runtimeVersion, _ := strings.CutSuffix(it.Status.RuntimeVersion, "-SNAPSHOT")
		if versionNumber, err := strconv.Atoi(strings.ReplaceAll(runtimeVersion, ".", "")); err == nil {
			return versionNumber >= 380 // >= runtimeVersion 3.8.0
		} else {
			return runtimeVersion >= "3.8.0"
		}
	}

	return false
}

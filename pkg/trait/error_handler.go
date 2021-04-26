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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// The error-handler is a platform trait used to inject Error Handler source into the integration runtime.
//
// +camel-k:trait=error-handler
type errorHandlerTrait struct {
	BaseTrait `property:",squash"`
}

func newErrorHandlerTrait() Trait {
	return &errorHandlerTrait{
		BaseTrait: NewBaseTrait("error-handler", 500),
	}
}

// IsPlatformTrait overrides base class method
func (t *errorHandlerTrait) IsPlatformTrait() bool {
	return true
}

func (t *errorHandlerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	return e.Integration.Spec.GetConfigurationProperty(v1alpha1.ErrorHandlerRefName) != "", nil
}

func (t *errorHandlerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		err := addErrorHandlerAsSource(e)
		if err != nil {
			return err
		}
	}
	return nil
}

func addErrorHandlerAsSource(e *Environment) error {
	errorHandlerRefName := e.Integration.Spec.GetConfigurationProperty(v1alpha1.ErrorHandlerRefName)
	// TODO change to yaml flow when we fix https://issues.apache.org/jira/browse/CAMEL-16486
	errorHandlerSource := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "ErrorHandlerSource.java",
			Content: fmt.Sprintf(`
			import org.apache.camel.builder.RouteBuilder;
			public class ErrorHandlerSource extends RouteBuilder {
			@Override
			public void configure() throws Exception {
				errorHandler("%s");
			  }
			}
			`, errorHandlerRefName),
		},
		Language: v1.LanguageJavaSource,
		Type:     v1.SourceTypeErrorHandler,
	}

	e.Integration.Status.AddOrReplaceGeneratedSources(errorHandlerSource)

	return nil
}

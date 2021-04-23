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
	"encoding/json"
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
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

	return e.Integration.Spec.ErrorHandler.Type != "", nil
}

func (t *errorHandlerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if e.Integration.Spec.ErrorHandler.Type != "" {
			// Possible error handler
			err := addErrorHandlerAsSource(e)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func addErrorHandlerAsSource(e *Environment) error {
	errorHandlerStatement, err := parseErrorHandler(e)
	if err != nil {
		return err
	}

	// TODO change to yaml flow when we fix https://issues.apache.org/jira/browse/CAMEL-16486
	errorHandlerSource := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name: "ErrorHandlerSource.java",
			Content: fmt.Sprintf(`
			import org.apache.camel.builder.RouteBuilder;
			public class ErrorHandlerSource extends RouteBuilder {
			@Override
			public void configure() throws Exception {
			  %s
			  }
			}
			`, errorHandlerStatement),
		},
		Language: v1.LanguageJavaSource,
		Type:     v1.SourceTypeErrorHandler,
	}

	e.Integration.Status.AddOrReplaceGeneratedSources(errorHandlerSource)

	return nil
}

func addErrorHandlerBeanConfiguration(e *Environment, fqn string) error {
	// camel.beans.defaultErrorHandler = #class:the-full-qualified-class-name
	e.Integration.Status.AddConfigurationsIfMissing(v1.ConfigurationSpec{
		Type:  "property",
		Value: fmt.Sprintf("camel.beans.defaultErrorHandler=#class:%s", fqn),
	})
	return nil
}

func parseErrorHandler(e *Environment) (string, error) {
	errorHandlerSpec := e.Integration.Spec.ErrorHandler
	switch errorHandlerSpec.Type {
	case "none":
		return `errorHandler(noErrorHandler());`, nil
	case "log":
		errorHandlerConfiguration, err := parseErrorHandlerConfiguration(errorHandlerSpec.Parameters)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf(`errorHandler(defaultErrorHandler()%v);`, errorHandlerConfiguration), nil
	case "dead-letter-channel":
		errorHandlerConfiguration, err := parseErrorHandlerConfiguration(errorHandlerSpec.Parameters)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf(`errorHandler(deadLetterChannel("%v")%v);`, errorHandlerSpec.URI, errorHandlerConfiguration), nil
	case "ref":
		// TODO using URI temporarily, fix it properly
		return fmt.Sprintf(`errorHandler("%v");`, errorHandlerSpec.URI), nil
	case "bean":
		// TODO using URI temporarily, fix it properly
		addErrorHandlerBeanConfiguration(e, errorHandlerSpec.URI)
		return fmt.Sprintf(`errorHandler("%v");`, "defaultErrorHandler"), nil
	}

	return "", fmt.Errorf("Cannot recognize any error handler of type %s", errorHandlerSpec.Type)
}

func parseErrorHandlerConfiguration(conf *v1.ErrorHandlerParameters) (string, error) {
	javaPropertiesBuilder := ""
	var properties map[string]interface{}
	err := json.Unmarshal(conf.RawMessage, &properties)
	if err != nil {
		return "", err
	}
	for method, value := range properties {
		javaPropertiesBuilder = javaPropertiesBuilder + fmt.Sprintf(".%s(%v)\n", method, value)
	}

	return javaPropertiesBuilder, nil
}

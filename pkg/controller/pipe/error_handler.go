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

package pipe

import (
	"encoding/json"
	"errors"
	"fmt"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/bindings"
)

const defaultCamelErrorHandler = "defaultErrorHandler"

// maybeErrorHandler will return a Binding mapping a DeadLetterChannel, a Log or a None Error Handler.
// If the bindings has no URI, then, you can assume it's a none Error Handler.
func maybeErrorHandler(errHandlConf *v1.ErrorHandlerSpec, bindingContext bindings.BindingContext) (*bindings.Binding, error) {
	if errHandlConf == nil {
		return nil, nil
	}

	var errorHandlerBinding *bindings.Binding

	errorHandlerSpec, err := parseErrorHandler(&errHandlConf.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("could not parse error handler: %w", err)
	}
	// We need to get the translated URI from any referenced resource (ie, kamelets)
	if errorHandlerSpec.Type() == v1.ErrorHandlerTypeSink {
		errorHandlerBinding, err = bindings.Translate(
			bindingContext,
			bindings.EndpointContext{Type: v1.EndpointTypeErrorHandler},
			*errorHandlerSpec.Endpoint(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not determine error handler URI: %w", err)
		}
	} else {
		// Create a new binding otherwise in order to store error handler application properties
		errorHandlerBinding = &bindings.Binding{
			ApplicationProperties: make(map[string]string),
		}
		if errorHandlerSpec.Type() == v1.ErrorHandlerTypeLog {
			errorHandlerBinding.URI = defaultCamelErrorHandler
		}
	}

	err = setErrorHandlerConfiguration(errorHandlerBinding, errorHandlerSpec)
	if err != nil {
		return nil, fmt.Errorf("could not set integration error handler: %w", err)
	}

	return errorHandlerBinding, nil
}

func parseErrorHandler(rawMessage *v1.RawMessage) (v1.ErrorHandler, error) {
	var properties map[v1.ErrorHandlerType]v1.RawMessage
	err := json.Unmarshal(*rawMessage, &properties)
	if err != nil {
		return nil, err
	}
	if len(properties) > 1 {
		return nil, fmt.Errorf("you must provide just 1 error handler, provided %d", len(properties))
	}

	for errHandlType, errHandlValue := range properties {
		var dst v1.ErrorHandler
		switch errHandlType {
		case v1.ErrorHandlerTypeNone:
			dst = new(v1.ErrorHandlerNone)
		case v1.ErrorHandlerTypeLog:
			dst = new(v1.ErrorHandlerLog)
		case v1.ErrorHandlerTypeSink:
			dst = new(v1.ErrorHandlerSink)
		default:
			return nil, fmt.Errorf("unknown error handler type %s", errHandlType)
		}

		if err = json.Unmarshal(errHandlValue, dst); err != nil {
			return nil, err
		}
		if err = dst.Validate(); err != nil {
			return nil, err
		}

		return dst, nil
	}

	return nil, errors.New("you must provide any supported error handler")
}

func setErrorHandlerConfiguration(errorHandlerBinding *bindings.Binding, errorHandler v1.ErrorHandler) error {
	properties, err := errorHandler.Configuration()
	if err != nil {
		return err
	}
	// initialize map if not yet initialized
	if errorHandlerBinding.ApplicationProperties == nil {
		errorHandlerBinding.ApplicationProperties = make(map[string]string)
	}
	for key, value := range properties {
		errorHandlerBinding.ApplicationProperties[key] = fmt.Sprintf("%v", value)
	}

	return nil
}

// translateCamelErrorHandler will translate a binding as an error handler YAML as expected by Camel.
func translateCamelErrorHandler(b *bindings.Binding) map[string]any {
	yamlCode := map[string]any{}
	switch b.URI {
	case "":
		yamlCode["errorHandler"] = map[string]any{
			"noErrorHandler": map[string]any{},
		}
	case defaultCamelErrorHandler:
		yamlCode["errorHandler"] = map[string]any{
			"defaultErrorHandler": map[string]any{
				"logName": "err",
			},
		}
	default:
		yamlCode["errorHandler"] = map[string]any{
			"deadLetterChannel": map[string]any{
				"deadLetterUri": b.URI,
			},
		}
	}

	return yamlCode
}

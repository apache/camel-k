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

package kameletbinding

import (
	"encoding/json"
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/pkg/errors"
)

func maybeErrorHandler(errHandlConf v1alpha1.ErrorHandlerSpec, bindingContext bindings.BindingContext) (*bindings.Binding, error) {
	var errorHandlerBinding *bindings.Binding
	if errHandlConf.RawMessage != nil {
		errorHandlerSpec, err := parseErrorHandler(errHandlConf.RawMessage)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse error handler")
		}
		// We need to get the translated URI from any referenced resource (ie, kamelets)
		if errorHandlerSpec.Type() == v1alpha1.ErrorHandlerTypeDeadLetterChannel {
			errorHandlerBinding, err = bindings.Translate(bindingContext, bindings.EndpointContext{Type: v1alpha1.EndpointTypeErrorHandler}, *errorHandlerSpec.Endpoint())
			if err != nil {
				return nil, errors.Wrap(err, "could not determine error handler URI")
			}
		} else {
			// Create a new binding otherwise in order to store application properties
			errorHandlerBinding = &bindings.Binding{
				URI:                   "error-handler",
				ApplicationProperties: make(map[string]string),
			}
		}

		err = setErrorHandlerConfiguration(errorHandlerBinding, errorHandlerSpec)
		if err != nil {
			return nil, errors.Wrap(err, "could not set integration error handler")
		}

		return errorHandlerBinding, nil
	}
	return nil, nil
}

func parseErrorHandler(rawMessage v1.RawMessage) (v1alpha1.ErrorHandler, error) {
	var properties map[v1alpha1.ErrorHandlerType]v1.RawMessage
	err := json.Unmarshal(rawMessage, &properties)
	if err != nil {
		return nil, err
	}
	if len(properties) > 1 {
		return nil, errors.Errorf("You must provide just 1 error handler, provided %d", len(properties))
	}

	for errHandlType, errHandlValue := range properties {
		var dst v1alpha1.ErrorHandler
		switch errHandlType {
		case v1alpha1.ErrorHandlerTypeNone:
			dst = new(v1alpha1.ErrorHandlerNone)
		case v1alpha1.ErrorHandlerTypeLog:
			dst = new(v1alpha1.ErrorHandlerLog)
		case v1alpha1.ErrorHandlerTypeDeadLetterChannel:
			dst = new(v1alpha1.ErrorHandlerDeadLetterChannel)
		case v1alpha1.ErrorHandlerTypeRef:
			dst = new(v1alpha1.ErrorHandlerRef)
		case v1alpha1.ErrorHandlerTypeBean:
			dst = new(v1alpha1.ErrorHandlerBean)
		default:
			return nil, errors.Errorf("Unknown error handler type %s", errHandlType)
		}

		err := json.Unmarshal(errHandlValue, dst)
		if err != nil {
			return nil, err
		}

		return dst, nil
	}

	return nil, errors.New("You must provide any supported error handler")
}

func setErrorHandlerConfiguration(errorHandlerBinding *bindings.Binding, errorHandler v1alpha1.ErrorHandler) error {
	properties, err := errorHandler.Configuration()
	if err != nil {
		return err
	}
	for key, value := range properties {
		errorHandlerBinding.ApplicationProperties[key] = fmt.Sprintf("%v", value)
	}
	if errorHandler.Type() == v1alpha1.ErrorHandlerTypeDeadLetterChannel && errorHandlerBinding.URI != "" {
		errorHandlerBinding.ApplicationProperties[fmt.Sprintf("%s.deadLetterUri", v1alpha1.ErrorHandlerAppPropertiesPrefix)] = fmt.Sprintf("%v", errorHandlerBinding.URI)
	}
	return nil
}

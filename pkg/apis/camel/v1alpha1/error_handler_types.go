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
	"encoding/json"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// ErrorHandlerRefName --
const ErrorHandlerRefName = "camel.k.errorHandler.ref"

// ErrorHandlerRefDefaultName --
const ErrorHandlerRefDefaultName = "defaultErrorHandler"

// ErrorHandlerAppPropertiesPrefix --
const ErrorHandlerAppPropertiesPrefix = "camel.beans.defaultErrorHandler"

// ErrorHandlerSpec represents an unstructured object for an error handler
type ErrorHandlerSpec struct {
	v1.RawMessage `json:",omitempty"`
}

// ErrorHandlerParameters represent an unstructured object for error handler parameters
type ErrorHandlerParameters struct {
	v1.RawMessage `json:",omitempty"`
}

// BeanProperties represent an unstructured object properties to be set on a bean
type BeanProperties struct {
	v1.RawMessage `json:",omitempty"`
}

// +kubebuilder:object:generate=false

// ErrorHandler is a generic interface that represent any type of error handler specification
type ErrorHandler interface {
	Type() ErrorHandlerType
	Endpoint() *Endpoint
	Configuration() (map[string]interface{}, error)
}

type baseErrorHandler struct {
}

// Type --
func (e baseErrorHandler) Type() ErrorHandlerType {
	return errorHandlerTypeBase
}

// Endpoint --
func (e baseErrorHandler) Endpoint() *Endpoint {
	return nil
}

// Configuration --
func (e baseErrorHandler) Configuration() (map[string]interface{}, error) {
	return nil, nil
}

// ErrorHandlerNone --
type ErrorHandlerNone struct {
	baseErrorHandler
}

// Type --
func (e ErrorHandlerNone) Type() ErrorHandlerType {
	return ErrorHandlerTypeNone
}

// Configuration --
func (e ErrorHandlerNone) Configuration() (map[string]interface{}, error) {
	return map[string]interface{}{
		ErrorHandlerAppPropertiesPrefix: "#class:org.apache.camel.builder.NoErrorHandlerBuilder",
		ErrorHandlerRefName:             ErrorHandlerRefDefaultName,
	}, nil
}

// ErrorHandlerLog represent a default (log) error handler type
type ErrorHandlerLog struct {
	ErrorHandlerNone
	Parameters *ErrorHandlerParameters `json:"parameters,omitempty"`
}

// Type --
func (e ErrorHandlerLog) Type() ErrorHandlerType {
	return ErrorHandlerTypeLog
}

// Configuration --
func (e ErrorHandlerLog) Configuration() (map[string]interface{}, error) {
	properties, err := e.ErrorHandlerNone.Configuration()
	if err != nil {
		return nil, err
	}
	properties[ErrorHandlerAppPropertiesPrefix] = "#class:org.apache.camel.builder.DefaultErrorHandlerBuilder"

	if e.Parameters != nil {
		var parameters map[string]interface{}
		err := json.Unmarshal(e.Parameters.RawMessage, &parameters)
		if err != nil {
			return nil, err
		}
		for key, value := range parameters {
			properties[ErrorHandlerAppPropertiesPrefix+"."+key] = value
		}
	}

	return properties, nil
}

// ErrorHandlerSink represents a sink error handler type which behave like a dead letter channel
type ErrorHandlerSink struct {
	ErrorHandlerLog
	DLCEndpoint *Endpoint `json:"endpoint,omitempty"`
}

// Type --
func (e ErrorHandlerSink) Type() ErrorHandlerType {
	return ErrorHandlerTypeSink
}

// Endpoint --
func (e ErrorHandlerSink) Endpoint() *Endpoint {
	return e.DLCEndpoint
}

// Configuration --
func (e ErrorHandlerSink) Configuration() (map[string]interface{}, error) {
	properties, err := e.ErrorHandlerLog.Configuration()
	if err != nil {
		return nil, err
	}
	properties[ErrorHandlerAppPropertiesPrefix] = "#class:org.apache.camel.builder.DeadLetterChannelBuilder"

	return properties, err
}

// ErrorHandlerType --
type ErrorHandlerType string

const (
	errorHandlerTypeBase ErrorHandlerType = ""
	// ErrorHandlerTypeNone --
	ErrorHandlerTypeNone ErrorHandlerType = "none"
	// ErrorHandlerTypeLog --
	ErrorHandlerTypeLog ErrorHandlerType = "log"
	// ErrorHandlerTypeSink --
	ErrorHandlerTypeSink ErrorHandlerType = "sink"
	// ErrorHandlerTypeDeadLetterChannel Deprecated in favour of ErrorHandlerTypeSink
	ErrorHandlerTypeDeadLetterChannel ErrorHandlerType = "dead-letter-channel"
)

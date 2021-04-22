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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// ErrorHandler represents an unstructured object for an error handler
type ErrorHandler struct {
	v1.RawMessage `json:",omitempty"`
}

// ErrorHandlerProperties represent an unstructured object for error handler parameters
type ErrorHandlerProperties struct {
	v1.RawMessage `json:",inline"`
}

// AbstractErrorHandler is a generic interface that represent any type of error handler specification
type AbstractErrorHandler interface {
	Type() ErrorHandlerType
	Params() *ErrorHandlerProperties
	Endpoint() *Endpoint
}

// ErrorHandlerNone --
type ErrorHandlerNone struct {
}

// NewErrorHandlerNone represents a no (ignore) error handler type
func NewErrorHandlerNone() ErrorHandlerNone {
	return ErrorHandlerNone{}
}

// Type --
func (e ErrorHandlerNone) Type() ErrorHandlerType {
	return ErrorHandlerTypeNone
}

// Params --
func (e ErrorHandlerNone) Params() *ErrorHandlerProperties {
	return nil
}

// Endpoint --
func (e ErrorHandlerNone) Endpoint() *Endpoint {
	return nil
}

// ErrorHandlerLog represent a default (log) error handler type
type ErrorHandlerLog struct {
	Parameters *ErrorHandlerProperties `json:"parameters,omitempty"`
}

// Type --
func (e ErrorHandlerLog) Type() ErrorHandlerType {
	return ErrorHandlerTypeLog
}

// Params --
func (e ErrorHandlerLog) Params() *ErrorHandlerProperties {
	return e.Parameters
}

// Endpoint --
func (e ErrorHandlerLog) Endpoint() *Endpoint {
	return nil
}

// ErrorHandlerDeadLetterChannel represents a dead letter channel error handler type
type ErrorHandlerDeadLetterChannel struct {
	*ErrorHandlerLog
	DLCEndpoint *Endpoint `json:"endpoint,omitempty"`
}

// Type --
func (e ErrorHandlerDeadLetterChannel) Type() ErrorHandlerType {
	return ErrorHandlerTypeDeadLetterChannel
}

// Params --
func (e ErrorHandlerDeadLetterChannel) Params() *ErrorHandlerProperties {
	return e.Parameters
}

// Endpoint --
func (e ErrorHandlerDeadLetterChannel) Endpoint() *Endpoint {
	return e.DLCEndpoint
}

// ErrorHandlerType --
type ErrorHandlerType string

const (
	// ErrorHandlerTypeNone --
	ErrorHandlerTypeNone ErrorHandlerType = "none"
	// ErrorHandlerTypeLog --
	ErrorHandlerTypeLog ErrorHandlerType = "log"
	// ErrorHandlerTypeDeadLetterChannel --
	ErrorHandlerTypeDeadLetterChannel ErrorHandlerType = "dead-letter-channel"
)

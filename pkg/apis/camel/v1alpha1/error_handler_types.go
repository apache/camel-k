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

// ErrorHandlerSpec represents an unstructured object for an error handler
type ErrorHandlerSpec struct {
	v1.RawMessage `json:",omitempty"`
}

// ErrorHandlerParameters represent an unstructured object for error handler parameters
type ErrorHandlerParameters struct {
	v1.RawMessage `json:",inline"`
}

// BeanProperties represent an unstructured object properties to be set on a bean
type BeanProperties struct {
	v1.RawMessage `json:",inline"`
}

// ErrorHandler is a generic interface that represent any type of error handler specification
type ErrorHandler interface {
	Type() ErrorHandlerType
	Params() *ErrorHandlerParameters
	Endpoint() *Endpoint
	Ref() *string
	Bean() *string
}

type abstractErrorHandler struct {
}

// Type --
func (e abstractErrorHandler) Type() ErrorHandlerType {
	return errorHandlerTypeAbstract
}

// Params --
func (e abstractErrorHandler) Params() *ErrorHandlerParameters {
	return nil
}

// Endpoint --
func (e abstractErrorHandler) Endpoint() *Endpoint {
	return nil
}

// Ref --
func (e abstractErrorHandler) Ref() *string {
	return nil
}

// Ref --
func (e abstractErrorHandler) Bean() *string {
	return nil
}

// ErrorHandlerNone --
type ErrorHandlerNone struct {
	*abstractErrorHandler
}

// Type --
func (e ErrorHandlerNone) Type() ErrorHandlerType {
	return ErrorHandlerTypeNone
}

// ErrorHandlerLog represent a default (log) error handler type
type ErrorHandlerLog struct {
	*abstractErrorHandler
	Parameters *ErrorHandlerParameters `json:"parameters,omitempty"`
}

// Type --
func (e ErrorHandlerLog) Type() ErrorHandlerType {
	return ErrorHandlerTypeLog
}

// Params --
func (e ErrorHandlerLog) Params() *ErrorHandlerParameters {
	return e.Parameters
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

// Endpoint --
func (e ErrorHandlerDeadLetterChannel) Endpoint() *Endpoint {
	return e.DLCEndpoint
}

// ErrorHandlerRef represents a reference to an error handler builder available in the registry
type ErrorHandlerRef struct {
	*abstractErrorHandler
	string
}

// Type --
func (e ErrorHandlerRef) Type() ErrorHandlerType {
	return ErrorHandlerTypeRef
}

// Ref --
func (e ErrorHandlerRef) Ref() *string {
	s := string(e.string)
	return &s
}

// ErrorHandlerBean represents a bean error handler type
type ErrorHandlerBean struct {
	*ErrorHandlerLog
	BeanType   *string         `json:"type,omitempty"`
	Properties *BeanProperties `json:"properties,omitempty"`
}

// Type --
func (e ErrorHandlerBean) Type() ErrorHandlerType {
	return ErrorHandlerTypeBean
}

// Bean --
func (e ErrorHandlerBean) Bean() *string {
	return e.BeanType
}

// ErrorHandlerType --
type ErrorHandlerType string

const (
	errorHandlerTypeAbstract ErrorHandlerType = ""
	// ErrorHandlerTypeNone --
	ErrorHandlerTypeNone ErrorHandlerType = "none"
	// ErrorHandlerTypeLog --
	ErrorHandlerTypeLog ErrorHandlerType = "log"
	// ErrorHandlerTypeDeadLetterChannel --
	ErrorHandlerTypeDeadLetterChannel ErrorHandlerType = "dead-letter-channel"
	// ErrorHandlerTypeRef --
	ErrorHandlerTypeRef ErrorHandlerType = "ref"
	// ErrorHandlerTypeBean --
	ErrorHandlerTypeBean ErrorHandlerType = "bean"
)

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

const (
	// ErrorHandlerRefName the reference name to use when looking for an error handler
	ErrorHandlerRefName = "camel.k.errorHandler.ref"
	// ErrorHandlerRefDefaultName the default name of the error handler
	ErrorHandlerRefDefaultName = "defaultErrorHandler"
	// ErrorHandlerAppPropertiesPrefix the prefix used for the error handler bean
	ErrorHandlerAppPropertiesPrefix = "camel.beans.defaultErrorHandler"
)

// ErrorHandlerSpec represents an unstructured object for an error handler
type ErrorHandlerSpec struct {
	RawMessage `json:",omitempty"`
}

// ErrorHandlerParameters represent an unstructured object for error handler parameters
type ErrorHandlerParameters struct {
	RawMessage `json:",omitempty"`
}

// BeanProperties represent an unstructured object properties to be set on a bean
type BeanProperties struct {
	RawMessage `json:",omitempty"`
}

// ErrorHandlerType a type of error handler (ie, sink)
type ErrorHandlerType string

const (
	errorHandlerTypeBase ErrorHandlerType = ""
	// ErrorHandlerTypeNone used to ignore any error event
	ErrorHandlerTypeNone ErrorHandlerType = "none"
	// ErrorHandlerTypeLog used to log the event producing the error
	ErrorHandlerTypeLog ErrorHandlerType = "log"
	// ErrorHandlerTypeSink used to send the event to a further sink (for future processing). This was previously known as dead-letter-channel.
	ErrorHandlerTypeSink ErrorHandlerType = "sink"
	// ErrorHandlerTypeDeadLetterChannel used to send the event to a dead letter channel
	// Deprecated in favour of ErrorHandlerTypeSink
	ErrorHandlerTypeDeadLetterChannel ErrorHandlerType = "dead-letter-channel"
)

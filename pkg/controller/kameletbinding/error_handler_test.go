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
	"fmt"
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestParseErrorHandlerNoneDoesSucceed(t *testing.T) {
	noErrorHandler, err := parseErrorHandler(
		[]byte(`{"none": null}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeNone, noErrorHandler.Type())
	parameters, err := noErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.NoErrorHandlerBuilder",
		parameters[v1alpha1.ErrorHandlerAppPropertiesPrefix])
	assert.Equal(t, v1alpha1.ErrorHandlerRefDefaultName, parameters[v1alpha1.ErrorHandlerRefName])
}

func TestParseErrorHandlerLogDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": null}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
	parameters, err := logErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DefaultErrorHandlerBuilder",
		parameters[v1alpha1.ErrorHandlerAppPropertiesPrefix])
	assert.Equal(t, v1alpha1.ErrorHandlerRefDefaultName, parameters[v1alpha1.ErrorHandlerRefName])
}

func TestParseErrorHandlerLogWithParametersDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": {"parameters": {"param1": "value1", "param2": "value2"}}}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
	parameters, err := logErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DefaultErrorHandlerBuilder",
		parameters[v1alpha1.ErrorHandlerAppPropertiesPrefix])
	assert.Equal(t, "value1", parameters["camel.beans.defaultErrorHandler.param1"])
	assert.Equal(t, "value2", parameters["camel.beans.defaultErrorHandler.param2"])
	assert.Equal(t, v1alpha1.ErrorHandlerRefDefaultName, parameters[v1alpha1.ErrorHandlerRefName])
}

func TestParseErrorHandlerSinkDoesSucceed(t *testing.T) {
	fmt.Println("Test")
	sinkErrorHandler, err := parseErrorHandler(
		[]byte(`{"sink": {"endpoint": {"uri": "someUri"}}}`),
	)
	assert.Nil(t, err)
	assert.NotNil(t, sinkErrorHandler)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeSink, sinkErrorHandler.Type())
	assert.Equal(t, "someUri", *sinkErrorHandler.Endpoint().URI)
	parameters, err := sinkErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DeadLetterChannelBuilder",
		parameters[v1alpha1.ErrorHandlerAppPropertiesPrefix])
	assert.Equal(t, v1alpha1.ErrorHandlerRefDefaultName, parameters[v1alpha1.ErrorHandlerRefName])
}

func TestParseErrorHandlerSinkWithParametersDoesSucceed(t *testing.T) {
	sinkErrorHandler, err := parseErrorHandler(
		[]byte(`{
			"sink": {
				"endpoint": {
					"uri": "someUri"
					}, 
				"parameters": 
					{"param1": "value1", "param2": "value2"}
			}
		}`),
	)
	assert.Nil(t, err)
	assert.NotNil(t, sinkErrorHandler)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeSink, sinkErrorHandler.Type())
	assert.Equal(t, "someUri", *sinkErrorHandler.Endpoint().URI)
	parameters, err := sinkErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DeadLetterChannelBuilder",
		parameters[v1alpha1.ErrorHandlerAppPropertiesPrefix])
	assert.Equal(t, v1alpha1.ErrorHandlerRefDefaultName, parameters[v1alpha1.ErrorHandlerRefName])
	assert.Equal(t, "value1", parameters["camel.beans.defaultErrorHandler.param1"])
	assert.Equal(t, "value2", parameters["camel.beans.defaultErrorHandler.param2"])
}

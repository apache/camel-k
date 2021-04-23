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
	assert.Equal(t, "#class:org.apache.camel.builder.NoErrorHandlerBuilder", parameters["camel.beans.defaultErrorHandler"])
}

func TestParseErrorHandlerLogDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": null}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
	parameters, err := logErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DefaultErrorHandlerBuilder", parameters["camel.beans.defaultErrorHandler"])
}

func TestParseErrorHandlerLogWithParametersDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": {"parameters": {"param1": "value1", "param2": "value2"}}}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
	assert.NotNil(t, logErrorHandler.Params())
	parameters, err := logErrorHandler.Configuration()
	assert.Nil(t, err)
	assert.Equal(t, "#class:org.apache.camel.builder.DefaultErrorHandlerBuilder", parameters["camel.beans.defaultErrorHandler"])
	assert.Equal(t, "value1", parameters["camel.beans.defaultErrorHandler.param1"])
	assert.Equal(t, "value2", parameters["camel.beans.defaultErrorHandler.param2"])
}

func TestParseErrorHandlerDLCDoesSucceed(t *testing.T) {
	dlcErrorHandler, err := parseErrorHandler(
		[]byte(`{"dead-letter-channel": {"endpoint": {"uri": "someUri"}}}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeDeadLetterChannel, dlcErrorHandler.Type())
	assert.Equal(t, "someUri", *dlcErrorHandler.Endpoint().URI)
}

func TestParseErrorHandlerDLCWithParametersDoesSucceed(t *testing.T) {
	dlcErrorHandler, err := parseErrorHandler(
		[]byte(`{
			"dead-letter-channel": {
				"endpoint": {
					"uri": "someUri"
					}, 
				"parameters": 
					[{"param1": "value1"}]
			}
		}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeDeadLetterChannel, dlcErrorHandler.Type())
	assert.Equal(t, "someUri", *dlcErrorHandler.Endpoint().URI)
	assert.NotNil(t, dlcErrorHandler.Params())
}

func TestParseErrorHandlerBeanWithParametersDoesSucceed(t *testing.T) {
	beanErrorHandler, err := parseErrorHandler(
		[]byte(`{
			"bean": {
				"type": "com.acme.MyType", 
				"parameters": 
					[{"param1": "value1"}]
			}
		}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeBean, beanErrorHandler.Type())
	assert.Equal(t, "com.acme.MyType", *beanErrorHandler.Bean())
	assert.NotNil(t, beanErrorHandler.Params())
}

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
}

func TestParseErrorHandlerLogDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": null}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
}

func TestParseErrorHandlerLogWithParametersDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": {"parameters": [{"param1": "value1"}, {"param2": "value2"}]}}`),
	)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.ErrorHandlerTypeLog, logErrorHandler.Type())
	assert.NotNil(t, logErrorHandler.Params())
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

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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseErrorHandlerNoneDoesSucceed(t *testing.T) {
	noErrorHandler, err := parseErrorHandler(
		[]byte(`{"none": null}`),
	)
	require.NoError(t, err)
	assert.Equal(t, v1.ErrorHandlerTypeNone, noErrorHandler.Type())
	_, err = noErrorHandler.Configuration()
	require.NoError(t, err)
}

func TestParseErrorHandlerLogDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": null}`),
	)
	require.NoError(t, err)
	assert.Equal(t, v1.ErrorHandlerTypeLog, logErrorHandler.Type())
	_, err = logErrorHandler.Configuration()
	require.NoError(t, err)
}

func TestParseErrorHandlerLogWithParametersDoesSucceed(t *testing.T) {
	logErrorHandler, err := parseErrorHandler(
		[]byte(`{"log": {"parameters": {"param1": "value1", "param2": "value2"}}}`),
	)
	require.NoError(t, err)
	assert.Equal(t, v1.ErrorHandlerTypeLog, logErrorHandler.Type())
	_, err = logErrorHandler.Configuration()
	require.NoError(t, err)
}

func TestParseErrorHandlerSinkDoesSucceed(t *testing.T) {
	sinkErrorHandler, err := parseErrorHandler(
		[]byte(`{"sink": {"endpoint": {"uri": "someUri"}}}`),
	)
	require.NoError(t, err)
	assert.NotNil(t, sinkErrorHandler)
	assert.Equal(t, v1.ErrorHandlerTypeSink, sinkErrorHandler.Type())
	assert.Equal(t, "someUri", *sinkErrorHandler.Endpoint().URI)
	_, err = sinkErrorHandler.Configuration()
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.NotNil(t, sinkErrorHandler)
	assert.Equal(t, v1.ErrorHandlerTypeSink, sinkErrorHandler.Type())
	assert.Equal(t, "someUri", *sinkErrorHandler.Endpoint().URI)
	_, err = sinkErrorHandler.Configuration()
	require.NoError(t, err)
}

func TestParseErrorHandlerSinkFail(t *testing.T) {
	_, err := parseErrorHandler(
		[]byte(`{"sink": {"ref": {"uri": "someUri"}}}`),
	)
	require.Error(t, err)
	assert.Equal(t, "missing endpoint in Error Handler Sink", err.Error())
}

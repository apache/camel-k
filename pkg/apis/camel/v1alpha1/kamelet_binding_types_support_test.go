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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberConversion(t *testing.T) {
	props := map[string]interface{}{
		"string":  "str",
		"int32":   1000000,
		"int64":   int64(10000000000),
		"float32": float32(123.123),
		"float64": float64(1111123.123),
	}
	ser, err := json.Marshal(props)
	assert.NoError(t, err)
	ep := EndpointProperties{
		RawMessage: ser,
	}
	res, err := ep.GetPropertyMap()
	assert.NoError(t, err)
	assert.Equal(t, "str", res["string"])
	assert.Equal(t, "1000000", res["int32"])
	assert.Equal(t, "10000000000", res["int64"])
	assert.Equal(t, "123.123", res["float32"])
	assert.Equal(t, "1111123.123", res["float64"])
}

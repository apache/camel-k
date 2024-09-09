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

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractKamelets(t *testing.T) {
	assert.Equal(t, "", ExtractKamelet("fail"))
	assert.Equal(t, "my-test", ExtractKamelet("kamelet:my-test"))
	assert.Equal(t, "my-test", ExtractKamelet("kamelet:my-test?"))
	assert.Equal(t, "my-test", ExtractKamelet("kamelet:my-test?option=1"))
	assert.Equal(t, "my-test", ExtractKamelet("kamelet:my-test?option=1&opt2=2"))
	assert.Equal(t, "my-test?kameletVersion=v1", ExtractKamelet("kamelet:my-test?option=1&opt2=2&kameletVersion=v1"))
	assert.Equal(t, "my-test?kameletVersion=v1", ExtractKamelet("kamelet:my-test?kameletVersion=v1"))
}

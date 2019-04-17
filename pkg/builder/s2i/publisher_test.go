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

package s2i

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPReplacement(t *testing.T) {
	assert.Equal(t, "docker-registry.default.svc:5000/myproject/camel-k:1234", getImageWithOpenShiftHost("172.30.1.1:5000/myproject/camel-k:1234"))
	assert.Equal(t, "docker-registry.default.svc/myproject/camel-k:1234", getImageWithOpenShiftHost("172.30.1.1/myproject/camel-k:1234"))
	assert.Equal(t, "docker-registry.default.svc/myproject/camel-k:1234", getImageWithOpenShiftHost("10.0.0.1/myproject/camel-k:1234"))
	assert.Equal(t, "docker-registry.default.svc/camel-k", getImageWithOpenShiftHost("10.0.0.1/camel-k"))
	assert.Equal(t, "10.0.2.3.4/camel-k", getImageWithOpenShiftHost("10.0.2.3.4/camel-k"))
	assert.Equal(t, "gcr.io/camel-k/camel-k:latest", getImageWithOpenShiftHost("gcr.io/camel-k/camel-k:latest"))
	assert.Equal(t, "docker.io/camel-k:latest", getImageWithOpenShiftHost("docker.io/camel-k:latest"))
}

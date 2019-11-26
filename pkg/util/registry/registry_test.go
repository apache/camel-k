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

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuth_GenerateDockerConfig(t *testing.T) {
	a := Auth{
		Username: "nic",
		Registry: "docker.io",
	}
	conf, err := a.GenerateDockerConfig()
	assert.Nil(t, err)
	assert.Equal(t, `{"auths":{"https://index.docker.io/v1/":{"auth":"bmljOg=="}}}`, string(conf))

	a = Auth{
		Username: "nic",
		Password: "pass",
		Registry: "quay.io",
	}
	conf, err = a.GenerateDockerConfig()
	assert.Nil(t, err)
	assert.Equal(t, `{"auths":{"quay.io":{"auth":"bmljOnBhc3M="}}}`, string(conf))

	a = Auth{
		Username: "nic",
		Password: "pass",
		Server:   "quay.io",
		Registry: "docker.io",
	}
	conf, err = a.GenerateDockerConfig()
	assert.Nil(t, err)
	assert.Equal(t, `{"auths":{"quay.io":{"auth":"bmljOnBhc3M="}}}`, string(conf))
}

func TestAuth_Validate(t *testing.T) {
	assert.NotNil(t, Auth{
		Username: "nic",
	}.validate())

	assert.NotNil(t, Auth{
		Server: "quay.io",
	}.validate())

	assert.Nil(t, Auth{
		Username: "nic",
		Server:   "quay.io",
	}.validate())
}

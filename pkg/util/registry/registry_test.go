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
	"context"
	"os"
	"testing"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestMountSecretRegistryConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := Auth{
		Username: "nic",
		Registry: "docker.io",
	}
	conf, _ := a.GenerateDockerConfig()
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret1",
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			v1.DockerConfigJsonKey: conf,
		},
	}

	c, err := test.NewFakeClient(&namespace, &secret)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	registryConfigDir, err := MountSecretRegistryConfig(ctx, c, "test", "prefix-", "my-secret1")
	assert.Nil(t, err)
	assert.NotNil(t, registryConfigDir)
	dockerfileExists, _ := util.FileExists(registryConfigDir + "/config.json")
	assert.True(t, dockerfileExists)
	os.RemoveAll(registryConfigDir)
}

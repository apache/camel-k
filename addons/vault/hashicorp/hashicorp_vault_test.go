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

package hashicorp

import (
	"testing"

	"github.com/apache/camel-k/v2/pkg/util/test"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestHashicorpVaultTraitApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	hashicorp := NewHashicorpVaultTrait()
	secrets, _ := hashicorp.(*hashicorpVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Engine = "test"
	secrets.Token = "wwww.testx1234590"
	secrets.Host = "localhost"
	secrets.Port = "9091"
	secrets.Scheme = "http"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "test", e.ApplicationProperties["camel.vault.hashicorp.engine"])
	assert.Equal(t, "wwww.testx1234590", e.ApplicationProperties["camel.vault.hashicorp.token"])
	assert.Equal(t, "localhost", e.ApplicationProperties["camel.vault.hashicorp.host"])
	assert.Equal(t, "9091", e.ApplicationProperties["camel.vault.hashicorp.port"])
	assert.Equal(t, "http", e.ApplicationProperties["camel.vault.hashicorp.scheme"])
}

func TestHashicorpVaultTraitWithSecretApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret1",
		},
		Data: map[string][]byte{
			"hashicorp-vault-token": []byte("my-hashicorp-vault-token"),
		},
	})
	hashicorp := NewHashicorpVaultTrait()
	secrets, _ := hashicorp.(*hashicorpVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Engine = "test"
	secrets.Token = "secret:my-secret1/hashicorp-vault-token"
	secrets.Host = "localhost"
	secrets.Port = "9091"
	secrets.Scheme = "http"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "test", e.ApplicationProperties["camel.vault.hashicorp.engine"])
	assert.Equal(t, "my-hashicorp-vault-token", e.ApplicationProperties["camel.vault.hashicorp.token"])
	assert.Equal(t, "localhost", e.ApplicationProperties["camel.vault.hashicorp.host"])
	assert.Equal(t, "9091", e.ApplicationProperties["camel.vault.hashicorp.port"])
	assert.Equal(t, "http", e.ApplicationProperties["camel.vault.hashicorp.scheme"])
}

func TestHashicorpVaultTraitWithConfigMapApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-configmap1",
		},
		Data: map[string]string{
			"hashicorp-vault-token": "my-hashicorp-vault-token",
		},
	})
	hashicorp := NewHashicorpVaultTrait()
	secrets, _ := hashicorp.(*hashicorpVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.Engine = "test"
	secrets.Token = "configmap:my-configmap1/hashicorp-vault-token"
	secrets.Host = "localhost"
	secrets.Port = "9091"
	secrets.Scheme = "http"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Empty(t, e.ApplicationProperties["quarkus.jaeger.enabled"])
	assert.Equal(t, "test", e.ApplicationProperties["camel.vault.hashicorp.engine"])
	assert.Equal(t, "my-hashicorp-vault-token", e.ApplicationProperties["camel.vault.hashicorp.token"])
	assert.Equal(t, "localhost", e.ApplicationProperties["camel.vault.hashicorp.host"])
	assert.Equal(t, "9091", e.ApplicationProperties["camel.vault.hashicorp.port"])
	assert.Equal(t, "http", e.ApplicationProperties["camel.vault.hashicorp.scheme"])
}

func createEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error), objects ...runtime.Object) *trait.Environment {
	t.Helper()

	catalog, err := catalogGen()
	client, _ := test.NewFakeClient(objects...)
	assert.Nil(t, err)

	e := trait.Environment{
		CamelCatalog:          catalog,
		ApplicationProperties: make(map[string]string),
		Client:                client,
	}

	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}
	platform := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test",
		},
	}
	e.Integration = &it
	e.Platform = &platform
	return &e
}

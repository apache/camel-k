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

package azure

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"
)

func TestAzureKeyVaultTraitApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	azure := NewAzureKeyVaultTrait()
	secrets, _ := azure.(*azureKeyVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.TenantID = "tenant-id"
	secrets.ClientID = "client-id"
	secrets.ClientSecret = "secret"
	secrets.VaultName = "my-vault"
	ok, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "client-id", e.ApplicationProperties["camel.vault.azure.clientId"])
	assert.Equal(t, "secret", e.ApplicationProperties["camel.vault.azure.clientSecret"])
	assert.Equal(t, "tenant-id", e.ApplicationProperties["camel.vault.azure.tenantId"])
	assert.Equal(t, "my-vault", e.ApplicationProperties["camel.vault.azure.vaultName"])
}

func createEnvironment(t *testing.T, catalogGen func() (*camel.RuntimeCatalog, error)) *trait.Environment {
	t.Helper()

	catalog, err := catalogGen()
	assert.Nil(t, err)

	e := trait.Environment{
		CamelCatalog:          catalog,
		ApplicationProperties: make(map[string]string),
	}

	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}
	e.Integration = &it
	return &e
}

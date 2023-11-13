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

func TestAzureKeyVaultTraitApply(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog)
	azure := NewAzureKeyVaultTrait()
	secrets, _ := azure.(*azureKeyVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.TenantID = "tenant-id"
	secrets.ClientID = "client-id"
	secrets.ClientSecret = "secret"
	secrets.VaultName = "my-vault"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "client-id", e.ApplicationProperties["camel.vault.azure.clientId"])
	assert.Equal(t, "secret", e.ApplicationProperties["camel.vault.azure.clientSecret"])
	assert.Equal(t, "tenant-id", e.ApplicationProperties["camel.vault.azure.tenantId"])
	assert.Equal(t, "my-vault", e.ApplicationProperties["camel.vault.azure.vaultName"])
}

func TestAzureKeyVaultTraitApplyWithConfigmapAndRefresh(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-configmap1",
		},
		Data: map[string]string{
			"azure-client-secret": "my-secret-key",
		},
	}, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-configmap2",
		},
		Data: map[string]string{
			"azure-storage-blob-key": "my-access-key",
		},
	})
	azure := NewAzureKeyVaultTrait()
	secrets, _ := azure.(*azureKeyVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.TenantID = "tenant-id"
	secrets.ClientID = "client-id"
	secrets.ClientSecret = "configmap:my-configmap1/azure-client-secret"
	secrets.VaultName = "my-vault"
	secrets.RefreshEnabled = pointer.Bool(true)
	secrets.BlobAccessKey = "configmap:my-configmap2/azure-storage-blob-key"
	secrets.BlobAccountName = "camel-k"
	secrets.BlobContainerName = "camel-k-container"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "client-id", e.ApplicationProperties["camel.vault.azure.clientId"])
	assert.Equal(t, "my-secret-key", e.ApplicationProperties["camel.vault.azure.clientSecret"])
	assert.Equal(t, "tenant-id", e.ApplicationProperties["camel.vault.azure.tenantId"])
	assert.Equal(t, "my-vault", e.ApplicationProperties["camel.vault.azure.vaultName"])
	assert.Equal(t, "camel-k", e.ApplicationProperties["camel.vault.azure.blobAccountName"])
	assert.Equal(t, "camel-k-container", e.ApplicationProperties["camel.vault.azure.blobContainerName"])
	assert.Equal(t, "my-access-key", e.ApplicationProperties["camel.vault.azure.blobAccessKey"])
	assert.True(t, true, e.ApplicationProperties["camel.vault.azure.refreshEnabled"])
}

func TestAzureKeyVaultTraitApplyWithSecretAndRefresh(t *testing.T) {
	e := createEnvironment(t, camel.QuarkusCatalog, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret1",
		},
		Data: map[string][]byte{
			"azure-client-secret": []byte("my-secret-key"),
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret2",
		},
		Data: map[string][]byte{
			"azure-storage-blob-key": []byte("my-access-key"),
		},
	})
	azure := NewAzureKeyVaultTrait()
	secrets, _ := azure.(*azureKeyVaultTrait)
	secrets.Enabled = pointer.Bool(true)
	secrets.TenantID = "tenant-id"
	secrets.ClientID = "client-id"
	secrets.ClientSecret = "secret:my-secret1/azure-client-secret"
	secrets.VaultName = "my-vault"
	secrets.RefreshEnabled = pointer.Bool(true)
	secrets.BlobAccessKey = "secret:my-secret2/azure-storage-blob-key"
	secrets.BlobAccountName = "camel-k"
	secrets.BlobContainerName = "camel-k-container"
	ok, condition, err := secrets.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = secrets.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "client-id", e.ApplicationProperties["camel.vault.azure.clientId"])
	assert.Equal(t, "my-secret-key", e.ApplicationProperties["camel.vault.azure.clientSecret"])
	assert.Equal(t, "tenant-id", e.ApplicationProperties["camel.vault.azure.tenantId"])
	assert.Equal(t, "my-vault", e.ApplicationProperties["camel.vault.azure.vaultName"])
	assert.Equal(t, "camel-k", e.ApplicationProperties["camel.vault.azure.blobAccountName"])
	assert.Equal(t, "camel-k-container", e.ApplicationProperties["camel.vault.azure.blobContainerName"])
	assert.Equal(t, "my-access-key", e.ApplicationProperties["camel.vault.azure.blobAccessKey"])
	assert.True(t, true, e.ApplicationProperties["camel.vault.azure.refreshEnabled"])
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

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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"k8s.io/utils/pointer"
)

// The Azure Key Vault trait can be used to use secrets from Azure Key Vault service
//
// The Azure Key Vault trait is disabled by default.
//
// For more information about how to use secrets from Azure Key Vault component take a look at the components docs: xref:components::azure-key-vault-component.adoc[Azure Key Vault component]
//
// A sample execution of this trait, would require
// the following trait options:
// -t azure-key-vault.enabled=true -t azure-key-vault.tenant-id="tenant-id" -t azure-key-vault.client-id="client-id" -t azure-key-vault.client-secret="client-secret" -t azure-key-vault.vault-name="vault-name"
//
// +camel-k:trait=azure-key-vault.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The Azure Tenant Id for accessing Key Vault
	TenantID string `property:"tenant-id,omitempty"`
	// The Azure Client Id for accessing Key Vault
	ClientID string `property:"client-id,omitempty"`
	// The Azure Client Secret for accessing Key Vault
	ClientSecret string `property:"client-secret,omitempty"`
	// The Azure Vault Name for accessing Key Vault
	VaultName string `property:"vault-name,omitempty"`
}

type azureKeyVaultTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

func NewAzureKeyVaultTrait() trait.Trait {
	return &azureKeyVaultTrait{
		BaseTrait: trait.NewBaseTrait("azure-key-vault", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *azureKeyVaultTrait) Configure(environment *trait.Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil
	}

	return true, nil
}

func (t *azureKeyVaultTrait) Apply(environment *trait.Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityAzureKeyVault)
		// Add the Camel Quarkus Azure Key Vault dependency
		util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-azure-key-vault")
	}

	if environment.IntegrationInRunningPhases() {
		environment.ApplicationProperties["camel.vault.azure.tenantId"] = t.TenantID
		environment.ApplicationProperties["camel.vault.azure.clientId"] = t.ClientID
		environment.ApplicationProperties["camel.vault.azure.clientSecret"] = t.ClientSecret
		environment.ApplicationProperties["camel.vault.azure.vaultName"] = t.VaultName
	}

	return nil
}

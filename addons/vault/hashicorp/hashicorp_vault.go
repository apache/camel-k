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
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// The Hashicorp Vault trait can be used to use secrets from Hashicorp Vault
//
// The Hashicorp Vault trait is disabled by default.
//
// For more information about how to use secrets from Hashicorp vault take a look at the components docs: xref:components::hashicorp-vault-component.adoc[Hashicorp Vault component]
//
// A sample execution of this trait, would require
// the following trait options:
// -t hashicorp-vault.enabled=true -t hashicorp-vault.token="token" -t hashicorp-vault.port="port" -t hashicorp-vault.engine="engine" -t hashicorp-vault.port="port" -t hashicorp-vault.scheme="scheme"
//
// WARNING: The trait is **deprecated** and will removed in future release versions: configure directly the Camel properties as required by the component instead.
//
// +camel-k:trait=hashicorp-vault.
// +camel-k:deprecated=2.5.0.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The Host to use
	Host string `property:"host" json:"host,omitempty"`
	// The Port to use
	Port string `property:"port" json:"port,omitempty"`
	// The Hashicorp engine to use
	Engine string `property:"engine" json:"engine,omitempty"`
	// The token to access Hashicorp Vault. This could be a plain text or a configmap/secret
	// The content of the hashicorp vault token is expected to be a text containing a valid Hashicorp Vault Token.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name, key optionally represents the resource key to be filtered (default key value = hashicorp-vault-token).
	Token string `property:"token" json:"token,omitempty"`
	// The scheme to access Hashicorp Vault
	Scheme string `property:"scheme" json:"scheme,omitempty"`
}

type hashicorpVaultTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

func NewHashicorpVaultTrait() trait.Trait {
	return &hashicorpVaultTrait{
		BaseTrait: trait.NewBaseTrait("hashicorp-vault", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *hashicorpVaultTrait) Configure(environment *trait.Environment) (bool, *trait.TraitCondition, error) {
	if environment.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	condition := trait.NewIntegrationCondition(
		"HashicorpVault",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		trait.TraitConfigurationReason,
		"HashicorpVault trait is deprecated and may be removed in future version: "+
			"configure directly the Camel properties as required by the component instead",
	)

	return true, condition, nil
}

func (t *hashicorpVaultTrait) Apply(environment *trait.Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityHashicorpVault)
	}

	if !environment.IntegrationInRunningPhases() {
		return nil
	}

	hits := v1.PlainConfigSecretRegexp.FindAllStringSubmatch(t.Token, -1)
	if len(hits) >= 1 {
		var res, _ = v1.DecodeValueSource(t.Token, "hashicorp-vault-token")

		secretValue, err := kubernetes.ResolveValueSource(environment.Ctx, environment.Client, environment.Platform.Namespace, &res)
		if err != nil {
			return err
		}
		if secretValue != "" {
			environment.ApplicationProperties["camel.vault.hashicorp.token"] = secretValue
		}
	} else {
		environment.ApplicationProperties["camel.vault.hashicorp.token"] = t.Token
	}

	environment.ApplicationProperties["camel.vault.hashicorp.host"] = t.Host
	environment.ApplicationProperties["camel.vault.hashicorp.port"] = t.Port
	environment.ApplicationProperties["camel.vault.hashicorp.engine"] = t.Engine
	environment.ApplicationProperties["camel.vault.hashicorp.scheme"] = t.Scheme

	return nil
}

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
	"regexp"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"k8s.io/utils/pointer"
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
// +camel-k:trait=hashicorp-vault.
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
	if environment.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	return true, nil, nil
}

func (t *hashicorpVaultTrait) Apply(environment *trait.Environment) error {
	rex := regexp.MustCompile(`^(configmap|secret):([a-zA-Z0-9][a-zA-Z0-9-]*)(/([a-zA-Z0-9].*))?$`)
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityHashicorpVault)
		// Deprecated
		// remove dependencies after Camel K Runtime > 2.16.0 and no longer supported or LTS
		util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-hashicorp-vault")
	}

	if environment.IntegrationInRunningPhases() {
		hits := rex.FindAllStringSubmatch(t.Token, -1)
		if len(hits) >= 1 {
			var res, _ = v1.DecodeValueSource(t.Token, "hashicorp-vault-token", "The Hashicorp Vault Token provided is not valid")
			if secretValue, err := kubernetes.ResolveValueSource(environment.Ctx, environment.Client, environment.Platform.Namespace, &res); err != nil {
				return err
			} else if secretValue != "" {
				environment.ApplicationProperties["camel.vault.hashicorp.token"] = string([]byte(secretValue))
			}
		} else {
			environment.ApplicationProperties["camel.vault.hashicorp.token"] = t.Token
		}
		environment.ApplicationProperties["camel.vault.hashicorp.host"] = t.Host
		environment.ApplicationProperties["camel.vault.hashicorp.port"] = t.Port
		environment.ApplicationProperties["camel.vault.hashicorp.engine"] = t.Engine
		environment.ApplicationProperties["camel.vault.hashicorp.scheme"] = t.Scheme
	}

	return nil
}

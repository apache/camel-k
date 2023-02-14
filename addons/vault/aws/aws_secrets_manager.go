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

package aws

import (
	"strconv"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"k8s.io/utils/pointer"
)

// The Secrets Manager trait can be used to use secrets from AWS Secrets Manager
//
// The AWS Secrets Manager trait is disabled by default.
//
// For more information about how to use secrets from AWS Secrets Manager take a look at the components docs: xref:components::aws-secrets-manager-component.adoc[AWS Secrets Manager component]
//
// A sample execution of this trait, would require
// the following trait options:
// -t aws-secrets-manager.enabled=true -t aws-secrets-manager.access-key="aws-access-key" -t aws-secrets-manager.secret-key="aws-secret-key" -t aws-secrets-manager.region="aws-region"
//
// To enable the automatic context reload on secrets updates you should define
// the following trait options:
// -t aws-secrets-manager.enabled=true -t aws-secrets-manager.access-key="aws-access-key" -t aws-secrets-manager.secret-key="aws-secret-key" -t aws-secrets-manager.region="aws-region" -t aws-secrets-manager.context-reload-enabled="true" -t aws-secrets-manager.refresh-enabled="true" -t aws-secrets-manager.refresh-period="30000" -t aws-secrets-manager.secrets="test*"
//
//
// +camel-k:trait=aws-secrets-manager.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The AWS Access Key to use
	AccessKey string `property:"access-key,omitempty"`
	// The AWS Secret Key to use
	SecretKey string `property:"secret-key,omitempty"`
	// The AWS Region to use
	Region string `property:"region,omitempty"`
	// Define if we want to use the Default Credentials Provider chain as authentication method
	UseDefaultCredentialsProvider *bool `property:"use-default-credentials-provider,omitempty"`
	// Define if we want to use the Camel Context Reload feature or not
	ContextReloadEnabled *bool `property:"context-reload-enabled,omitempty"`
	// Define if we want to use the Refresh Feature for secrets
	RefreshEnabled *bool `property:"refresh-enabled,omitempty"`
	// If Refresh is enabled, this defines the interval to check the refresh event
	RefreshPeriod string `property:"refresh-period,omitempty"`
	// If Refresh is enabled, the regular expression representing the secrets we want to track
	Secrets string `property:"refresh-period,omitempty"`
}

type awsSecretsManagerTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

func NewAwsSecretsManagerTrait() trait.Trait {
	return &awsSecretsManagerTrait{
		BaseTrait: trait.NewBaseTrait("aws-secrets-manager", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *awsSecretsManagerTrait) Configure(environment *trait.Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil
	}

	if t.UseDefaultCredentialsProvider == nil {
		t.UseDefaultCredentialsProvider = pointer.Bool(false)
	}
	if t.ContextReloadEnabled == nil {
		t.ContextReloadEnabled = pointer.Bool(false)
	}
	if t.RefreshEnabled == nil {
		t.RefreshEnabled = pointer.Bool(false)
	}

	return true, nil
}

func (t *awsSecretsManagerTrait) Apply(environment *trait.Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityAwsSecretsManager)
		// Add the Camel Quarkus AWS Secrets Manager
		util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-aws-secrets-manager")
	}

	if environment.IntegrationInRunningPhases() {
		environment.ApplicationProperties["camel.vault.aws.accessKey"] = t.AccessKey
		environment.ApplicationProperties["camel.vault.aws.secretKey"] = t.SecretKey
		environment.ApplicationProperties["camel.vault.aws.region"] = t.Region
		environment.ApplicationProperties["camel.vault.aws.defaultCredentialsProvider"] = strconv.FormatBool(*t.UseDefaultCredentialsProvider)
		environment.ApplicationProperties["camel.vault.aws.refreshEnabled"] = strconv.FormatBool(*t.RefreshEnabled)
		environment.ApplicationProperties["camel.main.context-reload-enabled"] = strconv.FormatBool(*t.ContextReloadEnabled)
		environment.ApplicationProperties["camel.vault.aws.refreshPeriod"] = t.RefreshPeriod
		if t.Secrets != "" {
			environment.ApplicationProperties["camel.vault.aws.secrets"] = t.Secrets
		}
	}

	return nil
}

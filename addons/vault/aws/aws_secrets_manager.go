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
	"regexp"
	"strconv"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
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
// +camel-k:trait=aws-secrets-manager.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The AWS Access Key to use. This could be a plain text or a configmap/secret
	// The content of the aws access key is expected to be a text containing a valid AWS access key.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name, key optionally represents the resource key to be filtered (default key value = aws-access-key).
	AccessKey string `property:"access-key" json:"accessKey,omitempty"`
	// The AWS Secret Key to use. This could be a plain text or a configmap/secret
	// The content of the aws secret key is expected to be a text containing a valid AWS secret key.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name, key optionally represents the resource key to be filtered (default key value = aws-secret-key).
	SecretKey string `property:"secret-key" json:"secretKey,omitempty"`
	// The AWS Region to use
	Region string `property:"region" json:"region,omitempty"`
	// Define if we want to use the Default Credentials Provider chain as authentication method
	UseDefaultCredentialsProvider *bool `property:"use-default-credentials-provider" json:"useDefaultCredentialsProvider,omitempty"`
	// Define if we want to use the Camel Context Reload feature or not
	ContextReloadEnabled *bool `property:"context-reload-enabled" json:"contextReloadEnabled,omitempty"`
	// Define if we want to use the Refresh Feature for secrets
	RefreshEnabled *bool `property:"refresh-enabled" json:"refreshEnabled,omitempty"`
	// If Refresh is enabled, this defines the interval to check the refresh event
	RefreshPeriod string `property:"refresh-period" json:"refreshPeriod,omitempty"`
	// If Refresh is enabled, the regular expression representing the secrets we want to track
	Secrets string `property:"secrets" json:"secrets,omitempty"`
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

func (t *awsSecretsManagerTrait) Configure(environment *trait.Environment) (bool, *trait.TraitCondition, error) {
	if environment.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil, nil
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

	return true, nil, nil
}

func (t *awsSecretsManagerTrait) Apply(environment *trait.Environment) error {
	rex := regexp.MustCompile(`^(configmap|secret):([a-zA-Z0-9][a-zA-Z0-9-]*)(/([a-zA-Z0-9].*))?$`)
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityAwsSecretsManager)
		// Deprecated
		// remove dependencies after Camel K Runtime > 2.16.0 and no longer supported or LTS
		util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-aws-secrets-manager")
	}

	if environment.IntegrationInRunningPhases() {
		hits := rex.FindAllStringSubmatch(t.AccessKey, -1)
		if len(hits) >= 1 {
			var res, _ = v1.DecodeValueSource(t.AccessKey, "aws-access-key", "The access Key provided is not valid")
			if secretValue, err := kubernetes.ResolveValueSource(environment.Ctx, environment.Client, environment.Platform.Namespace, &res); err != nil {
				return err
			} else if secretValue != "" {
				environment.ApplicationProperties["camel.vault.aws.accessKey"] = string([]byte(secretValue))
			}
		} else {
			environment.ApplicationProperties["camel.vault.aws.accessKey"] = t.AccessKey
		}
		hits = rex.FindAllStringSubmatch(t.SecretKey, -1)
		if len(hits) >= 1 {
			var res, _ = v1.DecodeValueSource(t.SecretKey, "aws-secret-key", "The secret Key provided is not valid")
			if secretValue, err := kubernetes.ResolveValueSource(environment.Ctx, environment.Client, environment.Platform.Namespace, &res); err != nil {
				return err
			} else if secretValue != "" {
				environment.ApplicationProperties["camel.vault.aws.secretKey"] = string([]byte(secretValue))
			}
		} else {
			environment.ApplicationProperties["camel.vault.aws.secretKey"] = t.SecretKey
		}
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

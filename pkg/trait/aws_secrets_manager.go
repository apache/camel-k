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

package trait

import (
	"strconv"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"k8s.io/utils/ptr"
)

type awsSecretsManagerTrait struct {
	BaseTrait
	traitv1.AwsSecretsManagerTrait `property:",squash"`
}

func newAwsSecretsManagerTrait() Trait {
	return &awsSecretsManagerTrait{
		BaseTrait: NewBaseTrait("aws-secrets-manager", TraitOrderBeforeControllerCreation),
	}
}

func (t *awsSecretsManagerTrait) Configure(environment *Environment) (bool, *TraitCondition, error) {
	if environment.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if t.UseDefaultCredentialsProvider == nil {
		t.UseDefaultCredentialsProvider = ptr.To(false)
	}
	if t.ContextReloadEnabled == nil {
		t.ContextReloadEnabled = ptr.To(false)
	}
	if t.RefreshEnabled == nil {
		t.RefreshEnabled = ptr.To(false)
	}

	return true, nil, nil
}

func (t *awsSecretsManagerTrait) Apply(environment *Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityAwsSecretsManager)
	}

	if !environment.IntegrationInRunningPhases() {
		return nil
	}

	hits := v1.PlainConfigSecretRegexp.FindAllStringSubmatch(t.AccessKey, -1)
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
	hits = v1.PlainConfigSecretRegexp.FindAllStringSubmatch(t.SecretKey, -1)
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

	return nil
}

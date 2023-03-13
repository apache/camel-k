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

package gcp

import (
	"strconv"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"k8s.io/utils/pointer"
)

// The Google Secret Manager trait can be used to use secrets from Google Secret Manager
//
// The Google Secret Manager trait is disabled by default.
//
// For more information about how to use secrets from Google Secret Manager take a look at the components docs: xref:components::google-secret-manager-component.adoc[AWS Secrets Manager component]
//
// A sample execution of this trait, would require
// the following trait options:
// -t gpc-secret-manager.enabled=true -t gpc-secret-manager.project-id="project-id" -t gpc-secret-manager.service-account-key="file:serviceaccount.json"
//
// To enable the automatic context reload on secrets updates you should define
// the following trait options:
// -t gpc-secret-manager.enabled=true -t gpc-secret-manager.project-id="project-id" -t gpc-secret-manager.service-account-key="file:serviceaccount.json" -t gcp-secret-manager.subscription-name="pubsub-sub" -t gcp-secret-manager.context-reload-enabled="true" -t gcp-secret-manager.refresh-enabled="true" -t gcp-secret-manager.refresh-period="30000" -t gcp-secret-manager.secrets="test*"
//
// +camel-k:trait=gcp-secret-manager.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The Project Id from Google Cloud
	ProjectID string `property:"project-id" json:"projectId,omitempty"`
	// The Path to a service account Key File to use secrets from Google Secret Manager
	ServiceAccountKey string `property:"service-account-key" json:"serviceAccountKey,omitempty"`
	// Define if we want to use the Default Instance approach for accessing the Google Secret Manager service
	UseDefaultInstance *bool `property:"use-default-instance" json:"useDefaultInstance,omitempty"`
	// Define if we want to use the Camel Context Reload feature or not
	ContextReloadEnabled *bool `property:"context-reload-enabled" json:"contextReloadEnabled,omitempty"`
	// Define if we want to use the Refresh Feature for secrets
	RefreshEnabled *bool `property:"refresh-enabled" json:"refreshEnabled,omitempty"`
	// If Refresh is enabled, this defines the interval to check the refresh event
	RefreshPeriod string `property:"refresh-period" json:"refreshPeriod,omitempty"`
	// If Refresh is enabled, the regular expression representing the secrets we want to track
	Secrets string `property:"secrets" json:"secrets,omitempty"`
	// If Refresh is enabled, this defines the subscription name to the Google PubSub topic used to keep track of updates
	SubscriptionName string `property:"subscription-name" json:"subscriptionName,omitempty"`
}

type gcpSecretManagerTrait struct {
	trait.BaseTrait
	Trait `property:",squash"`
}

func NewGcpSecretManagerTrait() trait.Trait {
	return &gcpSecretManagerTrait{
		BaseTrait: trait.NewBaseTrait("gcp-secret-manager", trait.TraitOrderBeforeControllerCreation),
	}
}

func (t *gcpSecretManagerTrait) Configure(environment *trait.Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	if !environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !environment.IntegrationInRunningPhases() {
		return false, nil
	}

	if t.UseDefaultInstance == nil {
		t.UseDefaultInstance = pointer.Bool(false)
	}

	if t.ContextReloadEnabled == nil {
		t.ContextReloadEnabled = pointer.Bool(false)
	}

	if t.RefreshEnabled == nil {
		t.RefreshEnabled = pointer.Bool(false)
	}

	return true, nil
}

func (t *gcpSecretManagerTrait) Apply(environment *trait.Environment) error {
	if environment.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&environment.Integration.Status.Capabilities, v1.CapabilityGcpSecretManager)
		// Add the Camel Quarkus Google Secrets Manager dependency
		util.StringSliceUniqueAdd(&environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-google-secret-manager")
	}

	if environment.IntegrationInRunningPhases() {
		environment.ApplicationProperties["camel.vault.gcp.projectId"] = t.ProjectID
		environment.ApplicationProperties["camel.vault.gcp.serviceAccountKey"] = t.ServiceAccountKey
		environment.ApplicationProperties["camel.vault.gcp.useDefaultInstance"] = strconv.FormatBool(*t.UseDefaultInstance)
		environment.ApplicationProperties["camel.vault.gcp.refreshEnabled"] = strconv.FormatBool(*t.RefreshEnabled)
		environment.ApplicationProperties["camel.main.context-reload-enabled"] = strconv.FormatBool(*t.ContextReloadEnabled)
		environment.ApplicationProperties["camel.vault.gcp.refreshPeriod"] = t.RefreshPeriod
		environment.ApplicationProperties["camel.vault.gcp.subscriptionName"] = t.SubscriptionName
		if t.Secrets != "" {
			environment.ApplicationProperties["camel.vault.gcp.secrets"] = t.Secrets
		}
	}

	return nil
}

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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
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
// +camel-k:trait=gcp-secret-manager.
type Trait struct {
	traitv1.Trait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The Project Id from Google Cloud
	ProjectID string `property:"project-id,omitempty"`
	// The Path to a service account Key File to use secrets from Google Secret Manager
	ServiceAccountKey string `property:"service-account-key,omitempty"`
	// Define if we want to use the Default Instance approach for accessing the Google Secret Manager service
	UseDefaultInstance *bool `property:"use-default-instance,omitempty"`
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
	}

	return nil
}

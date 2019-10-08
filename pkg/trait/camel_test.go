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
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestConfigureEnabledCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest()

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureDisabledCamelTraitFails(t *testing.T) {
	trait, environment := createNominalCamelTest()
	trait.Enabled = new(bool)

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.False(t, configured)
}

func TestApplyCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest()

	err := trait.Apply(environment)
	assert.Nil(t, err)
	assert.Equal(t, "0.0.1", environment.RuntimeVersion)
	assert.Equal(t, "1.23.0", environment.Integration.Status.CamelVersion)
	assert.Equal(t, "0.0.1", environment.Integration.Status.RuntimeVersion)
	assert.Equal(t, "1.23.0", environment.IntegrationKit.Status.CamelVersion)
	assert.Equal(t, "0.0.1", environment.IntegrationKit.Status.RuntimeVersion)
}

func TestApplyCamelTraitWithoutEnvironmentCatalogAndUnmatchableVersionFails(t *testing.T) {
	trait, environment := createNominalCamelTest()
	environment.CamelCatalog = nil
	environment.Integration.Status.CamelVersion = "Unmatchable version"

	err := trait.Apply(environment)
	assert.NotNil(t, err)
	assert.Equal(t, "unable to find catalog for: Unmatchable version", err.Error())
}

func TestCamelTraitGenerateMavenProjectSucceeds(t *testing.T) {
	trait, _ := createNominalCamelTest()

	mvnProject, err := trait.GenerateMavenProject("1.23.0", "1.0.0")
	assert.Nil(t, err)
	assert.NotNil(t, mvnProject)
	assert.Equal(t, "org.apache.camel.k.integration", mvnProject.GroupID)
	assert.Equal(t, "camel-k-catalog-generator", mvnProject.ArtifactID)
	assert.NotNil(t, mvnProject.Build)
	assert.Equal(t, "generate-resources", mvnProject.Build.DefaultGoal)
	assert.NotNil(t, mvnProject.Build.Plugins)
	assert.Len(t, mvnProject.Build.Plugins, 1)
	assert.Equal(t, "org.apache.camel.k", mvnProject.Build.Plugins[0].GroupID)
	assert.Equal(t, "camel-k-maven-plugin", mvnProject.Build.Plugins[0].ArtifactID)
	assert.NotNil(t, mvnProject.Build.Plugins[0].Executions)
	assert.Len(t, mvnProject.Build.Plugins[0].Executions, 1)
	assert.Equal(t, "generate-catalog", mvnProject.Build.Plugins[0].Executions[0].ID)
	assert.NotNil(t, mvnProject.Build.Plugins[0].Executions[0].Goals)
	assert.Len(t, mvnProject.Build.Plugins[0].Executions[0].Goals, 1)
	assert.Equal(t, "generate-catalog", mvnProject.Build.Plugins[0].Executions[0].Goals[0])
	assert.NotNil(t, mvnProject.Build.Plugins[0].Dependencies)
	assert.Len(t, mvnProject.Build.Plugins[0].Dependencies, 1)
	assert.Equal(t, "org.apache.camel", mvnProject.Build.Plugins[0].Dependencies[0].GroupID)
	assert.Equal(t, "camel-catalog", mvnProject.Build.Plugins[0].Dependencies[0].ArtifactID)
}

func createNominalCamelTest() (*camelTrait, *Environment) {

	client, _ := test.NewFakeClient()

	trait := newCamelTrait()
	enabled := true
	trait.Enabled = &enabled

	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1alpha1.CamelCatalogSpec{
				Version: "1.23.0",
			},
		},
		C:      context.TODO(),
		Client: client,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Status: v1alpha1.IntegrationStatus{
				CamelVersion:   "1.23.0",
				RuntimeVersion: "0.0.1",
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{},
	}

	return trait, environment
}

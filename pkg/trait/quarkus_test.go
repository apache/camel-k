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
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
)

func TestConfigureQuarkusTraitBuildSubmitted(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseBuildSubmitted

	configured, err := quarkusTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)

	build := getBuilderTask(environment.BuildTasks)
	assert.NotNil(t, t, build)

	assert.Len(t, build.Steps, len(builder.Quarkus.CommonSteps)+3)
}

func TestConfigureDisabledQuarkusTraitShouldFail(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	quarkusTrait.Enabled = pointer.Bool(false)

	configured, err := quarkusTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyQuarkusTraitDefaultKitLayout(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	configured, err := quarkusTrait.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)
	assert.Len(t, environment.IntegrationKits, 1)
	assert.Equal(t, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel], v1.IntegrationKitLayoutFastJar)
}

func TestApplyQuarkusTraitAnnotationKitConfiguration(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseBuildingKit

	v1.SetAnnotation(&environment.Integration.ObjectMeta, v1.TraitAnnotationPrefix+"quarkus.foo", "camel-k")

	configured, err := quarkusTrait.Configure(environment)
	assert.True(t, configured)
	assert.Nil(t, err)

	err = quarkusTrait.Apply(environment)
	assert.Nil(t, err)
	assert.Len(t, environment.IntegrationKits, 1)
	assert.Equal(t, v1.IntegrationKitLayoutFastJar, environment.IntegrationKits[0].Labels[v1.IntegrationKitLayoutLabel])
	assert.Equal(t, "camel-k", environment.IntegrationKits[0].Annotations[v1.TraitAnnotationPrefix+"quarkus.foo"])

}

func createNominalQuarkusTest() (*quarkusTrait, *Environment) {
	trait, _ := newQuarkusTrait().(*quarkusTrait)
	trait.Enabled = pointer.Bool(true)

	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{},
		BuildTasks: []v1.Task{
			{
				Builder: &v1.BuilderTask{},
			},
		},
		Platform: &v1.IntegrationPlatform{},
	}

	return trait, environment
}

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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/stretchr/testify/assert"
)

func TestConfigureQuarkusTraitShouldSucceed(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	configured, err := quarkusTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureDisabledQuarkusTraitShouldFail(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()
	quarkusTrait.Enabled = new(bool)

	configured, err := quarkusTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyQuarkusTraitDoesNothing(t *testing.T) {
	quarkusTrait, environment := createNominalQuarkusTest()

	err := quarkusTrait.Apply(environment)

	assert.Nil(t, err)
}

func TestQuarkusTraitAddBuildStepsShouldSucceed(t *testing.T) {
	quarkusTrait, _ := createNominalQuarkusTest()

	task := &v1.BuilderTask{
		Steps: builder.StepIDsFor(builder.DefaultSteps...),
	}

	quarkusTrait.addBuildSteps(task)

	assert.Len(t, task.Steps, 9)
}

func createNominalQuarkusTest() (*quarkusTrait, *Environment) {
	trait := newQuarkusTrait()
	enabled := true
	trait.Enabled = &enabled

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
		Platform: &v1.IntegrationPlatform{},
	}

	return trait, environment
}

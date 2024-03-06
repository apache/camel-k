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

	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestServiceBinding(t *testing.T) {
	sbTrait, environment := createNominalServiceBindingTest()
	sbTrait.Services = []string{
		"ConfigMap:default/my-service-name",
	}
	configured, condition, err := sbTrait.Configure(environment)

	assert.True(t, configured)
	require.NoError(t, err)
	assert.Nil(t, condition)

	// Required for local testing purposes only
	handlers = []pipeline.Handler{}
	err = sbTrait.Apply(environment)
	require.NoError(t, err)
	// TODO we should make the service binding trait to easily work with fake client
	// and test the apply result in the environment accordingly.
}

func createNominalServiceBindingTest() (*serviceBindingTrait, *Environment) {
	trait, _ := newServiceBindingTrait().(*serviceBindingTrait)
	client, _ := test.NewFakeClient()

	environment := &Environment{
		Client:       client,
		Catalog:      NewCatalog(client),
		CamelCatalog: &camel.RuntimeCatalog{},
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						Language: v1.LanguageJavaSource,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
		IntegrationKit: &v1.IntegrationKit{},
		Pipeline: []v1.Task{
			{
				Builder: &v1.BuilderTask{},
			},
			{
				Package: &v1.BuilderTask{},
			},
		},
		Platform: &v1.IntegrationPlatform{},
	}

	return trait, environment
}

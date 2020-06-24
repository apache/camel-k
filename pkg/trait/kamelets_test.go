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

	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestKameletsFinding(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:c1
    steps:
    - to: kamelet:c2
    - to: telegram:bots
    - to: kamelet://c0?prop=x
    - to: kamelet://complex-.-.-1a?prop=x&prop2
    - to: kamelet://complex-.-.-1b
    - to: kamelet:complex-.-.-1b
    - to: kamelet://complex-.-.-1b/a
    - to: kamelet://complex-.-.-1c/b
`)
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"c0", "c1", "c2", "complex-.-.-1a", "complex-.-.-1b", "complex-.-.-1c"}, trait.getKamelets())
}

func createKameletsTestEnvironment(flow string) (*kameletsTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()

	client, _ := test.NewFakeClient()
	trait := newKameletsTrait().(*kameletsTrait)
	trait.Ctx = context.TODO()
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(context.TODO(), nil),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "flow.yaml",
							Content: flow,
						},
						Language: v1.LanguageYaml,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}

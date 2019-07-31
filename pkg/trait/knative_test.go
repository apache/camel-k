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

	"github.com/apache/camel-k/pkg/util/envvar"

	"k8s.io/api/core/v1"

	"github.com/apache/camel-k/pkg/util"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestKnativeTraitWithCompressedSources(t *testing.T) {
	content := "H4sIAAAAAAAA/+JKK8rP1VAvycxNLbIqyUzOVtfkUlBQUNAryddQz8lPt8rMS8tX1+QCAAAA//8BAAD//3wZ4pUoAAAA"

	environment := Environment{
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile: v1alpha1.TraitProfileKnative,
				Sources: []v1alpha1.SourceSpec{
					{
						Language:    v1alpha1.LanguageJavaScript,
						Name:        "routes.js",
						Content:     content,
						Compression: true,
					},
				},
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        "registry",
				},
			},
		},
		EnvVars:        make([]v1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	err := NewCatalog().apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("knative")))
	assert.NotNil(t, envvar.Get(environment.EnvVars, "CAMEL_KNATIVE_CONFIGURATION"))

	services := 0
	environment.Resources.VisitKnativeService(func(service *serving.Service) {
		services++

		vars := service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env

		routes := util.LookupEnvVar(vars, "CAMEL_K_ROUTES")
		assert.NotNil(t, routes)
		assert.Equal(t, "env:CAMEL_K_ROUTE_000?language=js&compression=true", routes.Value)

		route := util.LookupEnvVar(vars, "CAMEL_K_ROUTE_000")
		assert.NotNil(t, route)
		assert.Equal(t, content, route.Value)
	})

	assert.True(t, services > 0)
	assert.True(t, environment.Resources.Size() > 0)
}

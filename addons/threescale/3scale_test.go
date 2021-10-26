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

package threescale

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestThreeScaleDisabled(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := &trait.Environment{
		CamelCatalog: catalog,
	}

	threeScale := NewThreeScaleTrait()
	enabled, err := threeScale.Configure(e)
	assert.Nil(t, err)
	assert.False(t, enabled)
}

func TestThreeScaleInjection(t *testing.T) {
	svc, e := createEnvironment(t)
	threeScale := NewThreeScaleTrait()
	enabled := true
	threeScale.(*threeScaleTrait).Enabled = &enabled
	ok, err := threeScale.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = threeScale.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "true", svc.Labels["discovery.3scale.net"])
	assert.Equal(t, "http", svc.Annotations["discovery.3scale.net/scheme"])
	assert.Equal(t, "/", svc.Annotations["discovery.3scale.net/path"])
	assert.Equal(t, "80", svc.Annotations["discovery.3scale.net/port"])
	assert.Equal(t, "/openapi.json", svc.Annotations["discovery.3scale.net/description-path"])
}

func TestThreeScaleInjectionNoAPIPath(t *testing.T) {
	svc, e := createEnvironment(t)
	threeScale := NewThreeScaleTrait()
	enabled := true
	threeScale.(*threeScaleTrait).Enabled = &enabled
	noPath := ""
	threeScale.(*threeScaleTrait).DescriptionPath = &noPath
	ok, err := threeScale.Configure(e)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = threeScale.Apply(e)
	assert.Nil(t, err)

	assert.Equal(t, "true", svc.Labels["discovery.3scale.net"])
	assert.Equal(t, "http", svc.Annotations["discovery.3scale.net/scheme"])
	assert.Equal(t, "/", svc.Annotations["discovery.3scale.net/path"])
	assert.Equal(t, "80", svc.Annotations["discovery.3scale.net/port"])
	_, p := svc.Annotations["discovery.3scale.net/description-path"]
	assert.False(t, p)
}

func createEnvironment(t *testing.T) (*corev1.Service, *trait.Environment) {
	t.Helper()

	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	e := trait.Environment{
		CamelCatalog: catalog,
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				v1.IntegrationLabel: "test",
			},
		},
	}
	e.Resources = kubernetes.NewCollection(&svc)

	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseDeploying,
		},
	}
	e.Integration = &it
	return &svc, &e
}

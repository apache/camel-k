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
	"os"
	"testing"
	"time"

	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestDslTraitApplicability(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	e := &Environment{
		CamelCatalog: catalog,
	}

	trait, _ := newOpenAPITrait().(*openAPITrait)
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration = &v1.Integration{
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseNone,
		},
	}
	enabled, condition, err = trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	trait.Configmaps = []string{"my-configmap"}

	enabled, condition, err = trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	enabled, condition, err = trait.Configure(e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
}

func TestRestDslTraitApplyError(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	fakeClient, _ := test.NewFakeClient()

	e := &Environment{
		CamelCatalog: catalog,
		Client:       fakeClient,
	}

	trait, _ := newOpenAPITrait().(*openAPITrait)
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration = &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}

	trait.Configmaps = []string{"my-configmap"}

	err = trait.Apply(e)
	require.Error(t, err)
	assert.Equal(t, "configmap my-configmap does not exist in namespace default", err.Error())
}

var openapi = `
{
  "swagger" : "2.0",
  "info" : {
    "version" : "1.0",
    "title" : "Greeting REST API"
  },
  "host" : "",
  "basePath" : "/camel/",
  "tags" : [ {
    "name" : "greetings",
    "description" : "Greeting to {name}"
  } ],
  "schemes" : [ "http" ],
  "paths" : {
    "/greetings/{name}" : {
      "get" : {
        "tags" : [ "greetings" ],
        "operationId" : "greeting-api",
        "parameters" : [ {
          "name" : "name",
          "in" : "path",
          "required" : true,
          "type" : "string"
        } ],
        "responses" : {
          "200" : {
            "description" : "Output type",
            "schema" : {
              "$ref" : "#/definitions/Greetings"
            }
          }
        }
      }
    }
  },
  "definitions" : {
    "Greetings" : {
      "type" : "object",
      "properties" : {
        "greetings" : {
          "type" : "string"
        }
      }
    }
  }
}
`

func TestRestDslTraitApplyWorks(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	fakeClient, _ := test.NewFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"greetings-api.json": openapi,
		},
	})

	e := &Environment{
		Ctx:          context.Background(),
		CamelCatalog: catalog,
		Client:       fakeClient,
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
			},
			Status: v1.IntegrationPlatformStatus{
				IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
					Cluster: v1.IntegrationPlatformClusterKubernetes,
					Build: v1.IntegrationPlatformBuildSpec{
						PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
						Registry:        v1.RegistrySpec{Address: "registry"},
						RuntimeVersion:  catalog.Runtime.Version,
						Maven:           v1.MavenSpec{},
						Timeout:         &metav1.Duration{Duration: time.Minute},
					},
				},
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	trait, _ := newOpenAPITrait().(*openAPITrait)
	trait.Client = fakeClient
	enabled, condition, err := trait.Configure(e)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)

	e.Integration = &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello",
			Namespace: "default",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}

	trait.Configmaps = []string{"my-configmap"}

	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}

	err = trait.Apply(e)
	require.NoError(t, err)

	assert.Equal(t, 1, e.Resources.Size())
	sourceCm := e.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == "hello-openapi-000"
	})
	assert.NotNil(t, sourceCm)
	assert.Contains(t, sourceCm.Data["content"], "get id=\"greeting-api\" path=\"/greetings/{name}")
}

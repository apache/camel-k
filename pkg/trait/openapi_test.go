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

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/internal"
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
	assert.NotNil(t, condition)
	assert.Equal(t, "OpenApi trait is deprecated and may be removed in future version: "+
		"use Camel REST contract first instead, https://camel.apache.org/manual/rest-dsl-openapi.html",
		condition.message,
	)
}

func TestRestDslTraitApplyError(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	fakeClient, _ := internal.NewFakeClient()

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
openapi: "3.0.0"
info:
  version: 1.0.0
  title: Swagger Petstore
  license:
    name: MIT
servers:
  - url: http://petstore.swagger.io/v1
paths:
  /pets:
    get:
      summary: List all pets
      operationId: listPets
      tags:
        - pets
      parameters:
        - name: limit
          in: query
          description: How many items to return at one time (max 100)
          required: false
          schema:
            type: integer
            format: int32
      responses:
        '200':
          description: A paged array of pets
          headers:
            x-next:
              description: A link to the next page of responses
              schema:
                type: string
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pets"
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
    post:
      summary: Create a pet
      operationId: createPets
      tags:
        - pets
      responses:
        '201':
          description: Null response
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
  /pets/{petId}:
    get:
      summary: Info for a specific pet
      operationId: showPetById
      tags:
        - pets
      parameters:
        - name: petId
          in: path
          required: true
          description: The id of the pet to retrieve
          schema:
            type: string
      responses:
        '200':
          description: Expected response to a valid request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
    Pets:
      type: array
      items:
        $ref: "#/components/schemas/Pet"
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
          format: int32
        message:
          type: string
`

func TestRestDslTraitApplyWorks(t *testing.T) {
	settings, err := maven.NewSettings(
		maven.Repositories(
			"https://repository.apache.org/content/groups/snapshots-group@id=apache@snapshots@noreleases",
		),
	)
	require.NoError(t, err)
	content, err := util.EncodeXML(settings)
	require.NoError(t, err)
	cm := newConfigMap("default", "maven-settings", "settings.xml", "settings.xml", string(content), nil)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)
	fakeClient, _ := internal.NewFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"pets.yaml": openapi,
		},
	},
		cm,
	)

	e := &Environment{
		Ctx:          context.Background(),
		CamelCatalog: catalog,
		Client:       fakeClient,
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "camel-k",
				Namespace: "default",
			},
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
						Maven: v1.MavenSpec{
							Settings: v1.ValueSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm.Name,
									},
									Key: "settings.xml",
								},
							},
						},
						Timeout: &metav1.Duration{Duration: time.Minute},
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
	assert.Contains(t, sourceCm.Data["content"], "get id=\"showPetById\" path=\"/pets/{petId}\"")
}

// newConfigMap will create a ConfigMap.
func newConfigMap(namespace, cmName, originalFilename string, generatedKey string,
	textData string, binaryData []byte) *corev1.ConfigMap {
	immutable := true
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
			Labels: map[string]string{
				kubernetes.ConfigMapOriginalFileNameLabel: originalFilename,
				kubernetes.ConfigMapAutogenLabel:          "true",
			},
		},
		Immutable: &immutable,
	}
	if textData != "" {
		cm.Data = map[string]string{
			generatedKey: textData,
		}
	}
	if binaryData != nil {
		cm.BinaryData = map[string][]byte{
			generatedKey: binaryData,
		}
	}
	return &cm
}

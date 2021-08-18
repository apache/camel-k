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

package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestMavenSettingsFromConfigMap(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string]string{
				"settings.xml": "setting-data",
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenSpec{
				Settings: v1.ValueSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "maven-settings",
						},
						Key: "settings.xml",
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.SettingsData)
}

func TestMavenSettingsFromSecret(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "maven-settings",
			},
			Data: map[string][]byte{
				"settings.xml": []byte("setting-data"),
			},
		},
	)

	assert.Nil(t, err)

	ctx := builderContext{
		Catalog:   catalog,
		Client:    c,
		Namespace: "ns",
		Build: v1.BuilderTask{
			Runtime: catalog.Runtime,
			Maven: v1.MavenSpec{
				Settings: v1.ValueSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "maven-settings",
						},
						Key: "settings.xml",
					},
				},
			},
		},
	}

	err = Project.GenerateProjectSettings.execute(&ctx)
	assert.Nil(t, err)

	assert.Equal(t, []byte("setting-data"), ctx.Maven.SettingsData)
}

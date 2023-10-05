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

package jib

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestJibMavenProfile(t *testing.T) {
	profile, err := JibMavenProfile("3.3.0", "0.2.0")

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(profile, "<profile>"))
	assert.True(t, strings.HasSuffix(profile, "</profile>"))
	assert.True(t, strings.Contains(profile, "<version>3.3.0</version>"))
	assert.True(t, strings.Contains(profile, "<version>0.2.0</version>"))

}

func TestJibMavenProfileDefaultValues(t *testing.T) {
	profile, err := JibMavenProfile("", "")

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(profile, "<profile>"))
	assert.True(t, strings.HasSuffix(profile, "</profile>"))
	assert.True(t, strings.Contains(profile, "<version>"+JibMavenPluginVersionDefault+"</version>"))
	assert.True(t, strings.Contains(profile, "<version>"+JibLayerFilterExtensionMavenVersionDefault+"</version>"))

}

func TestJibConfigMap(t *testing.T) {
	ctx := context.TODO()
	c, _ := test.NewFakeClient()
	kit := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			UID:       types.UID("8dc44a2b-063c-490e-ae02-1fab285ac70a"),
		},
		Status: v1.IntegrationKitStatus{
			Phase: v1.IntegrationKitPhaseBuildSubmitted,
		},
	}

	err := CreateProfileConfigmap(ctx, c, kit, "<profile>awesomeprofile</profile>")
	assert.NoError(t, err)

	key := ctrl.ObjectKey{
		Namespace: "ns",
		Name:      "test-publish-jib-profile",
	}
	cm := &corev1.ConfigMap{}
	err = c.Get(ctx, key, cm)
	assert.NoError(t, err)
	assert.Equal(t, cm.OwnerReferences[0].Name, "test")
	assert.Equal(t, cm.OwnerReferences[0].UID, types.UID("8dc44a2b-063c-490e-ae02-1fab285ac70a"))
	assert.NotNil(t, cm.Data["profile.xml"])
	assert.True(t, strings.Contains(cm.Data["profile.xml"], "awesome"))
}

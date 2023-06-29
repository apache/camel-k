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

package openshift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

var noSccAnnotationNamespace *corev1.Namespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "no-scc-annotations-namespace",
	},
}

var constrainedNamespace *corev1.Namespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "myuser",
		Annotations: map[string]string{
			"openshift.io/sa.scc.mcs":                 "s0:c26,c5",
			"openshift.io/sa.scc.supplemental-groups": "1000860000/10000",
			"openshift.io/sa.scc.uid-range":           "1000860000/10000",
		},
		Labels: map[string]string{
			"kubernetes.io/metadata.name":              "myuser",
			"pod-security.kubernetes.io/audit":         "restricted",
			"pod-security.kubernetes.io/audit-version": "v1.24",
			"pod-security.kubernetes.io/warn":          "restricted",
			"pod-security.kubernetes.io/warn-version":  "v1.24",
		},
	},
}

func TestGetUserIdNamespaceWithoutLabels(t *testing.T) {
	kclient := initClientWithNamespace(t, noSccAnnotationNamespace)

	_, errUID := GetOpenshiftUser(context.Background(), kclient, "no-scc-annotations-namespace")

	assert.NotNil(t, errUID)
	assert.Contains(t, errUID.Error(), "annotation 'openshift.io/sa.scc.uid-range' not found")
}

func TestGetUserIdNamespaceConstrained(t *testing.T) {
	kclient := initClientWithNamespace(t, constrainedNamespace)

	uid, errUID := GetOpenshiftUser(context.Background(), kclient, "myuser")

	assert.Nil(t, errUID)
	assert.Equal(t, "1000860000", uid)
}

func TestGetPodSecurityContextNamespaceWithoutLabels(t *testing.T) {
	kclient := initClientWithNamespace(t, noSccAnnotationNamespace)

	_, errPsc := GetOpenshiftPodSecurityContextRestricted(context.Background(), kclient, "no-scc-annotations-namespace")

	assert.NotNil(t, errPsc)
	assert.Contains(t, errPsc.Error(), "annotation 'openshift.io/sa.scc.uid-range' not found")
}

func TestGetPodSecurityContextNamespaceConstrained(t *testing.T) {
	kclient := initClientWithNamespace(t, constrainedNamespace)

	psc, errPsc := GetOpenshiftPodSecurityContextRestricted(context.Background(), kclient, "myuser")

	expectedFsGroup := int64(1000860000)
	assert.Nil(t, errPsc)
	assert.NotNil(t, psc)
	assert.Equal(t, expectedFsGroup, *psc.FSGroup)
}

func TestGetSecurityContextNamespaceWithoutLabels(t *testing.T) {
	kclient := initClientWithNamespace(t, noSccAnnotationNamespace)

	_, errSc := GetOpenshiftSecurityContextRestricted(context.Background(), kclient, "no-scc-annotations-namespace")

	assert.NotNil(t, errSc)
	assert.Contains(t, errSc.Error(), "annotation 'openshift.io/sa.scc.uid-range' not found")
}

func TestGetSecurityContextNamespaceConstrained(t *testing.T) {
	kclient := initClientWithNamespace(t, constrainedNamespace)

	sc, errSc := GetOpenshiftSecurityContextRestricted(context.Background(), kclient, "myuser")

	expectedUserID := int64(1000860000)
	assert.Nil(t, errSc)
	assert.NotNil(t, sc)
	assert.Equal(t, expectedUserID, *sc.RunAsUser)
}

func initClientWithNamespace(t *testing.T, ns *corev1.Namespace) *fakeclientset.Clientset {
	t.Helper()
	kclient := fakeclientset.NewSimpleClientset()
	_, err := kclient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	return kclient
}

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

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestOwner(t *testing.T) {
	env := SetUpOwnerEnvironment(t)

	processTestEnv(t, env)

	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("owner"))

	ValidateOwnerResources(t, env, true)
}

func SetUpOwnerEnvironment(t *testing.T) *Environment {
	t.Helper()

	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "camel:core")
	env.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"owner": test.TraitSpecFromMap(t, map[string]interface{}{
			"targetLabels":      []string{"com.mycompany/mylabel1"},
			"targetAnnotations": []string{"com.mycompany/myannotation2"},
		}),
	}

	env.Integration.SetLabels(map[string]string{
		"com.mycompany/mylabel1": "myvalue1",
		"com.mycompany/mylabel2": "myvalue2",
		"org.apache.camel/l1":    "l1",
	})
	env.Integration.SetAnnotations(map[string]string{
		"com.mycompany/myannotation1": "myannotation1",
		"com.mycompany/myannotation2": "myannotation2",
	})

	return env
}

func ValidateOwnerResources(t *testing.T, env *Environment, withOwnerRef bool) {
	t.Helper()

	assert.NotEmpty(t, env.Resources.Items())

	env.Resources.VisitMetaObject(func(res metav1.Object) {
		if withOwnerRef {
			assert.NotEmpty(t, res.GetOwnerReferences())
		} else {
			assert.Empty(t, res.GetOwnerReferences())
		}

		ValidateLabelsAndAnnotations(t, res)
	})

	deployments := make([]*appsv1.Deployment, 0)
	env.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		deployments = append(deployments, deployment)
	})

	assert.Len(t, deployments, 1)
	ValidateLabelsAndAnnotations(t, &deployments[0].Spec.Template)
}

func ValidateLabelsAndAnnotations(t *testing.T, res metav1.Object) {
	t.Helper()

	assert.Contains(t, res.GetLabels(), "com.mycompany/mylabel1")
	assert.Equal(t, "myvalue1", res.GetLabels()["com.mycompany/mylabel1"])

	assert.NotContains(t, res.GetLabels(), "com.mycompany/mylabel2")

	assert.Contains(t, res.GetAnnotations(), "com.mycompany/myannotation2")
	assert.Equal(t, "myannotation2", res.GetAnnotations()["com.mycompany/myannotation2"])
}

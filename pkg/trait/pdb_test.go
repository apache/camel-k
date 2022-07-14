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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigurePdbTraitDoesSucceed(t *testing.T) {
	pdbTrait, environment, _ := createPdbTest()
	configured, err := pdbTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigurePdbTraitDoesNotSucceed(t *testing.T) {
	pdbTrait, environment, _ := createPdbTest()

	pdbTrait.MinAvailable = "1"
	pdbTrait.MaxUnavailable = "2"
	configured, err := pdbTrait.Configure(environment)
	assert.NotNil(t, err)
	assert.False(t, configured)
}

func TestPdbIsCreatedWithoutParametersEnabled(t *testing.T) {
	pdbTrait, environment, _ := createPdbTest()

	pdb := pdbCreatedCheck(t, pdbTrait, environment)
	assert.Equal(t, int32(1), pdb.Spec.MaxUnavailable.IntVal)
}

func TestPdbIsCreatedWithMaxUnavailable(t *testing.T) {
	pdbTrait, environment, _ := createPdbTest()
	pdbTrait.MaxUnavailable = "1"

	pdb := pdbCreatedCheck(t, pdbTrait, environment)
	assert.Equal(t, int32(1), pdb.Spec.MaxUnavailable.IntVal)
}

func TestPdbIsCreatedWithMinAvailable(t *testing.T) {
	pdbTrait, environment, _ := createPdbTest()
	pdbTrait.MinAvailable = "2"

	pdb := pdbCreatedCheck(t, pdbTrait, environment)
	assert.Equal(t, int32(2), pdb.Spec.MinAvailable.IntVal)
}

func pdbCreatedCheck(t *testing.T, pdbTrait *pdbTrait, environment *Environment) *v1beta1.PodDisruptionBudget {
	t.Helper()

	err := pdbTrait.Apply(environment)
	assert.Nil(t, err)
	pdb := findPdb(environment.Resources)

	assert.NotNil(t, pdb)
	assert.Equal(t, environment.Integration.Name, pdb.Name)
	assert.Equal(t, environment.Integration.Namespace, pdb.Namespace)
	assert.Equal(t, environment.Integration.Labels, pdb.Labels)
	return pdb
}

func findPdb(resources *kubernetes.Collection) *v1beta1.PodDisruptionBudget {
	for _, a := range resources.Items() {
		if pdb, ok := a.(*v1beta1.PodDisruptionBudget); ok {
			return pdb
		}
	}
	return nil
}

// nolint: unparam
func createPdbTest() (*pdbTrait, *Environment, *appsv1.Deployment) {
	trait, _ := newPdbTrait().(*pdbTrait)
	trait.Enabled = pointer.Bool(true)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-name",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}

	environment := &Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(deployment),
	}

	return trait, environment, deployment
}

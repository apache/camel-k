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
	"encoding/json"
	"testing"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNominalDeploymentTraitTest() (*Environment, *appsv1.Deployment) {
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

	return environment, deployment
}

func createNominalMissingDeploymentTraitTest() *Environment {
	environment := &Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return environment
}

func createNominalKnativeServiceTraitTest() (*Environment, *serving.Service) {
	knativeService := &serving.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-name",
		},
		Spec: serving.ServiceSpec{},
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
		Resources: kubernetes.NewCollection(knativeService),
	}

	return environment, knativeService
}

func createNominalCronJobTraitTest() (*Environment, *batchv1.CronJob) {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "integration-name",
		},
		Spec: batchv1.CronJobSpec{},
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
		Resources: kubernetes.NewCollection(cronJob),
	}

	return environment, cronJob
}

// nolint: staticcheck
func configurationFromMap(t *testing.T, configMap map[string]interface{}) *traitv1.Configuration {
	t.Helper()

	data, err := json.Marshal(configMap)
	require.NoError(t, err)

	return &traitv1.Configuration{
		RawMessage: data,
	}
}

func traitToMap(t *testing.T, trait interface{}) map[string]interface{} {
	t.Helper()

	traitMap := make(map[string]interface{})

	data, err := json.Marshal(trait)
	require.NoError(t, err)

	err = json.Unmarshal(data, &traitMap)
	require.NoError(t, err)

	return traitMap
}

func ToAddonTrait(t *testing.T, config map[string]interface{}) v1.AddonTrait {
	t.Helper()

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var addon v1.AddonTrait
	err = json.Unmarshal(data, &addon)
	assert.NoError(t, err)

	return addon
}

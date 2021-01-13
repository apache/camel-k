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

	serving "knative.dev/serving/pkg/apis/serving/v1"

	appsv1 "k8s.io/api/apps/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func newTestProbesEnv(t *testing.T, provider v1.RuntimeProvider) Environment {
	var catalog *camel.RuntimeCatalog = nil
	var err error = nil

	switch provider {
	case v1.RuntimeProviderQuarkus:
		catalog, err = camel.QuarkusCatalog()
	default:
		panic("unknown provider " + provider)
	}

	assert.Nil(t, err)
	assert.NotNil(t, catalog)

	return Environment{
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
}

func newTestContainerTrait() *containerTrait {
	tr := newContainerTrait().(*containerTrait)
	tr.ProbesEnabled = util.BoolP(true)

	return tr
}

func TestProbesDepsQuarkus(t *testing.T) {
	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	ctr := newTestContainerTrait()

	ok, err := ctr.Configure(&env)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = ctr.Apply(&env)
	assert.Nil(t, err)
	assert.Contains(t, env.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-microprofile-health")
}

func TestProbesOnDeployment(t *testing.T) {
	target := appsv1.Deployment{}

	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	env.Resources.Add(&target)

	expose := true

	ctr := newTestContainerTrait()
	ctr.Expose = &expose
	ctr.LivenessTimeout = 1234

	err := ctr.Apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultProbePath, target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(defaultContainerPort), target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultProbePath, target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), target.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds)
}

func TestProbesOnDeploymentWithNoHttpPort(t *testing.T) {
	target := appsv1.Deployment{}

	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	env.Resources.Add(&target)

	ctr := newTestContainerTrait()
	ctr.PortName = "custom"
	ctr.LivenessTimeout = 1234

	err := ctr.Apply(&env)
	assert.Nil(t, err)
	assert.Nil(t, target.Spec.Template.Spec.Containers[0].LivenessProbe)
	assert.Nil(t, target.Spec.Template.Spec.Containers[0].ReadinessProbe)
}

func TestProbesOnKnativeService(t *testing.T) {
	target := serving.Service{}

	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	env.Resources.Add(&target)

	expose := true

	ctr := newTestContainerTrait()
	ctr.Expose = &expose
	ctr.LivenessTimeout = 1234

	err := ctr.Apply(&env)
	assert.Nil(t, err)

	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Host)
	assert.Equal(t, int32(0), target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultProbePath, target.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
	assert.Equal(t, "", target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Host)
	assert.Equal(t, int32(0), target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, defaultProbePath, target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), target.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds)
}

func TestProbesOnKnativeServiceWithNoHttpPort(t *testing.T) {
	target := serving.Service{}

	env := newTestProbesEnv(t, v1.RuntimeProviderQuarkus)
	env.Integration.Status.Phase = v1.IntegrationPhaseDeploying
	env.Resources.Add(&target)

	ctr := newTestContainerTrait()
	ctr.PortName = "custom"
	ctr.LivenessTimeout = 1234

	err := ctr.Apply(&env)
	assert.Nil(t, err)
	assert.Nil(t, target.Spec.Template.Spec.Containers[0].LivenessProbe)
	assert.Nil(t, target.Spec.Template.Spec.Containers[0].ReadinessProbe)
}

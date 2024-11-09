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
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestConfigureJolokiaTraitInRunningPhaseDoesSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	configured, condition, err := trait.Configure(environment)

	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)
}

func TestApplyJolokiaTraitNominalShouldSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()

	err := trait.Apply(environment)

	require.NoError(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)
	assert.NotNil(t, container)

	assert.Equal(t, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar=discoveryEnabled=false,host=*,port=8778",
		"-cp", "dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar",
	},
		container.Args)

	assert.Len(t, container.Ports, 1)
	containerPort := container.Ports[0]
	assert.Equal(t, "jolokia", containerPort.Name)
	assert.Equal(t, int32(8778), containerPort.ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, containerPort.Protocol)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionJolokiaAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestApplyJolokiaTraitForOpenShiftProfileShouldSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.IntegrationKit.Spec.Profile = v1.TraitProfileOpenShift

	err := trait.Apply(environment)

	require.NoError(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)
	assert.NotNil(t, container)

	assert.Equal(t, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar=caCert=/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt," +
			"clientPrincipal.1=cn=system:master-proxy,clientPrincipal.2=cn=hawtio-online.hawtio.svc," +
			"clientPrincipal.3=cn=fuse-console.fuse.svc,discoveryEnabled=false,extendedClientCheck=true," +
			"host=*,port=8778,protocol=https,useSslClientAuthentication=true",
		"-cp", "dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar"},
		container.Args,
	)

	assert.Len(t, container.Ports, 1)
	containerPort := container.Ports[0]
	assert.Equal(t, "jolokia", containerPort.Name)
	assert.Equal(t, int32(8778), containerPort.ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, containerPort.Protocol)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionJolokiaAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestApplyJolokiaTraitWithoutContainerShouldReportJolokiaUnavailable(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.Resources = kubernetes.NewCollection()

	err := trait.Apply(environment)

	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1.IntegrationConditionJolokiaAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionFalse, condition.Status)
}

func TestApplyJolokiaTraitWithOptionShouldOverrideDefault(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	trait.Options = []string{
		"host=explicit-host",
		"discoveryEnabled=true",
		"protocol=http",
		"caCert=.cacert",
		"extendedClientCheck=false",
		"clientPrincipal=cn:any",
		"useSslClientAuthentication=false",
	}

	err := trait.Apply(environment)

	require.NoError(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)

	assert.Equal(t, container.Args, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar=caCert=.cacert,clientPrincipal=cn:any," +
			"discoveryEnabled=true,extendedClientCheck=false,host=explicit-host,port=8778,protocol=http," +
			"useSslClientAuthentication=false",
		"-cp", "dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar",
	})
}

func TestApplyJolokiaTraitWithUnparseableOptionShouldReturnError(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	trait.Options = []string{"unparseable options"}

	err := trait.Apply(environment)

	require.Error(t, err)
}

func createNominalJolokiaTest() (*jolokiaTrait, *Environment) {
	trait, _ := newJolokiaTrait().(*jolokiaTrait)
	trait.Enabled = ptr.To(true)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Spec: v1.IntegrationKitSpec{
				Profile: v1.TraitProfileKubernetes,
			},
			Status: v1.IntegrationKitStatus{
				Artifacts: []v1.Artifact{
					{
						ID:     "org.jolokia.jolokia-agent-jvm-1.7.1.jar",
						Target: "dependencies/lib/main/org.jolokia.jolokia-agent-jvm-1.7.1.jar",
					},
				},
			},
		},
		Resources: kubernetes.NewCollection(
			&appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: defaultContainerName,
								},
							},
						},
					},
				},
			},
		),
	}

	return trait, environment
}

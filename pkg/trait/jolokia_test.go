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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestConfigureJolokiaTraitInRunningPhaseDoesSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseRunning

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestApplyJolokiaTraitNominalShouldSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()

	err := trait.Apply(environment)

	assert.Nil(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)
	assert.NotNil(t, container)

	assert.Equal(t, container.Args, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-jvm-1.7.0.jar=discoveryEnabled=false,host=*,port=8778",
	})

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

	assert.Nil(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)
	assert.NotNil(t, container)

	assert.Equal(t, container.Args, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-jvm-1.7.0.jar=caCert=/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt," +
			"clientPrincipal.1=cn=system:master-proxy,clientPrincipal.2=cn=hawtio-online.hawtio.svc," +
			"clientPrincipal.3=cn=fuse-console.fuse.svc,discoveryEnabled=false,extendedClientCheck=true," +
			"host=*,port=8778,protocol=https,useSslClientAuthentication=true",
	})

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

	assert.Nil(t, err)
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

	assert.Nil(t, err)

	container := environment.Resources.GetContainerByName(defaultContainerName)

	assert.Equal(t, container.Args, []string{
		"-javaagent:dependencies/lib/main/org.jolokia.jolokia-jvm-1.7.0.jar=caCert=.cacert,clientPrincipal=cn:any," +
			"discoveryEnabled=true,extendedClientCheck=false,host=explicit-host,port=8778,protocol=http," +
			"useSslClientAuthentication=false",
	})
}

func TestApplyJolokiaTraitWithUnparseableOptionShouldReturnError(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	trait.Options = []string{"unparseable options"}

	err := trait.Apply(environment)

	assert.NotNil(t, err)
}

func TestSetDefaultJolokiaOptionShouldNotOverrideOptionsMap(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{"key": "value"}
	optionValue := ""

	trait.setDefaultJolokiaOption(options, &optionValue, "key", "new-value")

	assert.Equal(t, "", optionValue)
}

func TestSetDefaultStringJolokiaOptionShouldSucceed(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	var option *string

	trait.setDefaultJolokiaOption(options, &option, "key", "new-value")

	assert.Equal(t, "new-value", *option)
}

func TestSetDefaultStringJolokiaOptionShouldNotOverrideExistingValue(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	optionValue := "existing-value"
	option := &optionValue

	trait.setDefaultJolokiaOption(options, &option, "key", "new-value")

	assert.Equal(t, "existing-value", *option)
}

func TestSetDefaultIntJolokiaOptionShouldSucceed(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	var option *int

	trait.setDefaultJolokiaOption(options, &option, "key", 2)

	assert.Equal(t, 2, *option)
}

func TestSetDefaultIntJolokiaOptionShouldNotOverrideExistingValue(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	optionValue := 1
	option := &optionValue

	trait.setDefaultJolokiaOption(options, &option, "key", 2)

	assert.Equal(t, 1, *option)
}

func TestSetDefaultBoolJolokiaOptionShouldSucceed(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	var option *bool

	trait.setDefaultJolokiaOption(options, &option, "key", true)

	assert.Equal(t, true, *option)
}

func TestSetDefaultBoolJolokiaOptionShouldNotOverrideExistingValue(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	option := BoolP(false)

	trait.setDefaultJolokiaOption(options, &option, "key", true)

	assert.Equal(t, false, *option)
}

func TestAddStringOptionToJolokiaOptions(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	optionValue := "value"

	trait.addToJolokiaOptions(options, "key", &optionValue)

	assert.Len(t, options, 1)
	assert.Equal(t, "value", options["key"])
}

func TestAddIntOptionToJolokiaOptions(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}

	trait.addToJolokiaOptions(options, "key", 1)

	assert.Len(t, options, 1)
	assert.Equal(t, "1", options["key"])
}

func TestAddIntPointerOptionToJolokiaOptions(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}
	optionValue := 1

	trait.addToJolokiaOptions(options, "key", &optionValue)

	assert.Len(t, options, 1)
	assert.Equal(t, "1", options["key"])
}

func TestAddBoolPointerOptionToJolokiaOptions(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}

	trait.addToJolokiaOptions(options, "key", BoolP(false))

	assert.Len(t, options, 1)
	assert.Equal(t, "false", options["key"])
}

func TestAddWrongTypeOptionToJolokiaOptionsDoesNothing(t *testing.T) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	options := map[string]string{}

	trait.addToJolokiaOptions(options, "key", new(rune))

	assert.Len(t, options, 0)
}

func createNominalJolokiaTest() (*jolokiaTrait, *Environment) {
	trait := newJolokiaTrait().(*jolokiaTrait)
	trait.Enabled = BoolP(true)

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

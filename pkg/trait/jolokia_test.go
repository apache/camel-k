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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestConfigureJolokiaTraitInRightPhaseDoesSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
	assert.Equal(t, *trait.Host, "*")
	assert.Equal(t, *trait.DiscoveryEnabled, false)
	assert.Nil(t, trait.Protocol)
	assert.Nil(t, trait.CaCert)
	assert.Nil(t, trait.ExtendedClientCheck)
	assert.Nil(t, trait.ClientPrincipal)
	assert.Nil(t, trait.UseSslClientAuthentication)
}

func TestConfigureJolokiaTraitInWrongPhaseDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.Integration.Status.Phase = v1alpha1.IntegrationPhaseRunning

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureJolokiaTraitWithUnparseableOptionsDoesNotSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	options := "unparseable csv"
	trait.Options = &options

	configured, err := trait.Configure(environment)
	assert.NotNil(t, err)
	assert.False(t, configured)
}

func TestConfigureJolokiaTraitForOpenShiftProfileShouldSetDefaultHttpsJolokiaOptions(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.IntegrationKit.Spec.Profile = v1alpha1.TraitProfileOpenShift

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
	assert.Equal(t, *trait.Protocol, "https")
	assert.Equal(t, *trait.CaCert, "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	assert.Equal(t, *trait.ExtendedClientCheck, true)
	assert.Equal(t, *trait.ClientPrincipal, "cn=system:master-proxy")
	assert.Equal(t, *trait.UseSslClientAuthentication, true)
}

func TestConfigureJolokiaTraitWithOptionsShouldPreventDefaultJolokiaOptions(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.IntegrationKit.Spec.Profile = v1alpha1.TraitProfileOpenShift
	options := "host=explicit-host," +
		"discoveryEnabled=true," +
		"protocol=http," +
		"caCert=.cacert," +
		"extendedClientCheck=false," +
		"clientPrincipal=cn:any," +
		"useSslClientAuthentication=false"
	trait.Options = &options

	configured, err := trait.Configure(environment)

	assert.Nil(t, err)
	assert.True(t, configured)
	assert.Nil(t, trait.Host)
	assert.Nil(t, trait.DiscoveryEnabled)
	assert.Nil(t, trait.Protocol)
	assert.Nil(t, trait.CaCert)
	assert.Nil(t, trait.ExtendedClientCheck)
	assert.Nil(t, trait.ClientPrincipal)
	assert.Nil(t, trait.UseSslClientAuthentication)
}

func TestApplyJolokiaTraitNominalShouldSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()

	err := trait.Apply(environment)

	container := environment.Resources.GetContainerByName(defaultContainerName)

	assert.Nil(t, err)
	test.EnvVarHasValue(t, container.Env, "AB_JOLOKIA_AUTH_OPENSHIFT", "false")
	test.EnvVarHasValue(t, container.Env, "AB_JOLOKIA_OPTS", "port=8778")
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	assert.NotNil(t, container)
	assert.Len(t, container.Ports, 1)
	containerPort := container.Ports[0]
	assert.Equal(t, "jolokia", containerPort.Name)
	assert.Equal(t, int32(8778), containerPort.ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, containerPort.Protocol)

	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1alpha1.IntegrationConditionJolokiaAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestApplyJolokiaTraitWithoutContainerShouldReportJolokiaUnavailable(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	environment.Resources = kubernetes.NewCollection()

	err := trait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)
	condition := environment.Integration.Status.Conditions[0]
	assert.Equal(t, v1alpha1.IntegrationConditionJolokiaAvailable, condition.Type)
	assert.Equal(t, corev1.ConditionFalse, condition.Status)
}

func TestApplyJolokiaTraitWithOptionShouldOverrideDefault(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	options := "host=explicit-host," +
		"discoveryEnabled=true," +
		"protocol=http," +
		"caCert=.cacert," +
		"extendedClientCheck=false," +
		"clientPrincipal=cn:any," +
		"useSslClientAuthentication=false"
	trait.Options = &options

	err := trait.Apply(environment)

	container := environment.Resources.GetContainerByName(defaultContainerName)

	assert.Nil(t, err)
	ev := envvar.Get(container.Env, "AB_JOLOKIA_OPTS")
	assert.NotNil(t, ev)
	assert.Contains(t, ev.Value, "port=8778", "host=explicit-host", "discoveryEnabled=true", "protocol=http", "caCert=.cacert")
	assert.Contains(t, ev.Value, "extendedClientCheck=false", "clientPrincipal=cn:any", "useSslClientAuthentication=false")
}

func TestApplyJolokiaTraitWithUnparseableOptionShouldReturnError(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	options := "unparseable options"
	trait.Options = &options

	err := trait.Apply(environment)

	assert.NotNil(t, err)
}

func TestApplyDisabledJolokiaTraitShouldNotSucceed(t *testing.T) {
	trait, environment := createNominalJolokiaTest()
	trait.Enabled = new(bool)

	err := trait.Apply(environment)

	container := environment.Resources.GetContainerByName(defaultContainerName)

	assert.Nil(t, err)
	test.EnvVarHasValue(t, container.Env, "AB_JOLOKIA_OFF", "true")
}

func TestSetDefaultJolokiaOptionShoudlNotOverrideOptionsMap(t *testing.T) {
	options := map[string]string{"key": "value"}
	optionValue := ""
	setDefaultJolokiaOption(options, &optionValue, "key", "new-value")
	assert.Equal(t, "", optionValue)
}

func TestSetDefaultStringJolokiaOptionShoudlSucceed(t *testing.T) {
	options := map[string]string{}
	var option *string
	setDefaultJolokiaOption(options, &option, "key", "new-value")
	assert.Equal(t, "new-value", *option)
}

func TestSetDefaultStringJolokiaOptionShoudlNotOverrideExistingValue(t *testing.T) {
	options := map[string]string{}
	optionValue := "existing-value"
	option := &optionValue
	setDefaultJolokiaOption(options, &option, "key", "new-value")
	assert.Equal(t, "existing-value", *option)
}

func TestSetDefaultIntJolokiaOptionShoudlSucceed(t *testing.T) {
	options := map[string]string{}
	var option *int
	setDefaultJolokiaOption(options, &option, "key", 2)
	assert.Equal(t, 2, *option)
}

func TestSetDefaultIntJolokiaOptionShoudlNotOverrideExistingValue(t *testing.T) {
	options := map[string]string{}
	optionValue := 1
	option := &optionValue
	setDefaultJolokiaOption(options, &option, "key", 2)
	assert.Equal(t, 1, *option)
}

func TestSetDefaultBoolJolokiaOptionShoudlSucceed(t *testing.T) {
	options := map[string]string{}
	var option *bool
	setDefaultJolokiaOption(options, &option, "key", true)
	assert.Equal(t, true, *option)
}

func TestSetDefaultBoolJolokiaOptionShoudlNotOverrideExistingValue(t *testing.T) {
	options := map[string]string{}
	option := new(bool)
	setDefaultJolokiaOption(options, &option, "key", true)
	assert.Equal(t, false, *option)
}

func TestAddStringOptionToJolokiaOptions(t *testing.T) {
	options := map[string]string{}
	optionValue := "value"

	addToJolokiaOptions(options, "key", &optionValue)

	assert.Len(t, options, 1)
	assert.Equal(t, "value", options["key"])
}

func TestAddIntOptionToJolokiaOptions(t *testing.T) {
	options := map[string]string{}

	addToJolokiaOptions(options, "key", 1)

	assert.Len(t, options, 1)
	assert.Equal(t, "1", options["key"])
}

func TestAddIntPointerOptionToJolokiaOptions(t *testing.T) {
	options := map[string]string{}
	optionValue := 1

	addToJolokiaOptions(options, "key", &optionValue)

	assert.Len(t, options, 1)
	assert.Equal(t, "1", options["key"])
}

func TestAddBoolPointerOptionToJolokiaOptions(t *testing.T) {
	options := map[string]string{}

	addToJolokiaOptions(options, "key", new(bool))

	assert.Len(t, options, 1)
	assert.Equal(t, "false", options["key"])
}

func TestAddWrongTypeOptionToJolokiaOptionsDoesNothing(t *testing.T) {
	options := map[string]string{}

	addToJolokiaOptions(options, "key", new(rune))

	assert.Len(t, options, 0)
}

func createNominalJolokiaTest() (*jolokiaTrait, *Environment) {
	trait := newJolokiaTrait()
	enabled := true
	trait.Enabled = &enabled

	environment := &Environment{
		Catalog: NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Spec: v1alpha1.IntegrationKitSpec{
				Profile: v1alpha1.TraitProfileKubernetes,
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

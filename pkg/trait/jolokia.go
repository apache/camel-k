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
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"

	corev1 "k8s.io/api/core/v1"
)

type jolokiaTrait struct {
	BaseTrait `property:",squash"`

	// Jolokia JVM agent configuration
	// See https://jolokia.org/reference/html/agents.html
	CaCert                     *string `property:"ca-cert"`
	ClientPrincipal            *string `property:"client-principal"`
	DiscoveryEnabled           *bool   `property:"discovery-enabled"`
	ExtendedClientCheck        *bool   `property:"extended-client-check"`
	Host                       *string `property:"host"`
	Password                   *string `property:"password"`
	Port                       int     `property:"port"`
	Protocol                   *string `property:"protocol"`
	User                       *string `property:"user"`
	UseSslClientAuthentication *bool   `property:"use-ssl-client-authentication"`

	// Extra configuration options
	Options *string `property:"options"`
}

// The Jolokia trait must be executed prior to the deployment trait
// as it mutates environment variables
func newJolokiaTrait() *jolokiaTrait {
	return &jolokiaTrait{
		BaseTrait: newBaseTrait("jolokia"),
		Port:      8778,
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, error) {
	options, err := parseCsvMap(t.Options)
	if err != nil {
		return false, err
	}

	setDefaultJolokiaOption(options, &t.Host, "host", "*")
	setDefaultJolokiaOption(options, &t.DiscoveryEnabled, "discoveryEnabled", false)

	// Configure HTTPS by default for OpenShift
	if e.DetermineProfile() == v1alpha1.TraitProfileOpenShift {
		setDefaultJolokiaOption(options, &t.Protocol, "protocol", "https")
		setDefaultJolokiaOption(options, &t.CaCert, "caCert", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
		setDefaultJolokiaOption(options, &t.ExtendedClientCheck, "extendedClientCheck", true)
		setDefaultJolokiaOption(options, &t.ClientPrincipal, "clientPrincipal", "cn=system:master-proxy")
		setDefaultJolokiaOption(options, &t.UseSslClientAuthentication, "useSslClientAuthentication", true)
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *jolokiaTrait) Apply(e *Environment) (err error) {
	if t.Enabled == nil || !*t.Enabled {
		// Deactivate the Jolokia Java agent
		// Note: the AB_JOLOKIA_OFF environment variable acts as an option flag
		envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_OFF", "true")
		return nil
	}

	// Need to set it explicitly as it default to true
	envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_AUTH_OPENSHIFT", "false")

	// Configure the Jolokia Java agent
	// Populate first with the extra options
	options, err := parseCsvMap(t.Options)
	if err != nil {
		return err
	}

	// Then add explicitly set trait configuration properties
	addToJolokiaOptions(options, "caCert", t.CaCert)
	addToJolokiaOptions(options, "clientPrincipal", t.ClientPrincipal)
	addToJolokiaOptions(options, "discoveryEnabled", t.DiscoveryEnabled)
	addToJolokiaOptions(options, "extendedClientCheck", t.ExtendedClientCheck)
	addToJolokiaOptions(options, "host", t.Host)
	addToJolokiaOptions(options, "password", t.Password)
	addToJolokiaOptions(options, "port", t.Port)
	addToJolokiaOptions(options, "protocol", t.Protocol)
	addToJolokiaOptions(options, "user", t.User)
	addToJolokiaOptions(options, "useSslClientAuthentication", t.UseSslClientAuthentication)

	// Lastly set the AB_JOLOKIA_OPTS environment variable from the fabric8/s2i-java base image
	optionValues := make([]string, 0, len(options))
	for k, v := range options {
		optionValues = append(optionValues, k+"="+v)
	}
	envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_OPTS", strings.Join(optionValues, ","))

	containerName := defaultContainerName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		containerName = dt.(*containerTrait).Name
	}

	container := e.Resources.GetContainerByName(containerName)
	if container == nil {
		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionJolokiaAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionContainerNotAvailableReason,
			"",
		)

		return nil
	}

	containerPort := corev1.ContainerPort{
		Name:          "jolokia",
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	e.Integration.Status.SetCondition(
		v1alpha1.IntegrationConditionJolokiaAvailable,
		corev1.ConditionTrue,
		v1alpha1.IntegrationConditionJolokiaAvailableReason,
		fmt.Sprintf("%s(%s/%d)", container.Name, containerPort.Name, containerPort.ContainerPort),
	)

	container.Ports = append(container.Ports, containerPort)

	return nil
}

func setDefaultJolokiaOption(options map[string]string, option interface{}, key string, value interface{}) {
	// Do not override existing option
	if _, ok := options[key]; ok {
		return
	}
	switch o := option.(type) {
	case **bool:
		if *o == nil {
			v := value.(bool)
			*o = &v
		}
	case **int:
		if *o == nil {
			v := value.(int)
			*o = &v
		}
	case **string:
		if *o == nil {
			v := value.(string)
			*o = &v
		}
	}
}

func addToJolokiaOptions(options map[string]string, key string, value interface{}) {
	switch v := value.(type) {
	case *bool:
		if v != nil {
			options[key] = strconv.FormatBool(*v)
		}
	case *int:
		if v != nil {
			options[key] = strconv.Itoa(*v)
		}
	case int:
		options[key] = strconv.Itoa(v)
	case *string:
		if v != nil {
			options[key] = *v
		}
	}
}

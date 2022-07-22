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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util"
)

type jolokiaTrait struct {
	BaseTrait
	traitv1.JolokiaTrait `property:",squash"`
}

func newJolokiaTrait() Trait {
	return &jolokiaTrait{
		BaseTrait: NewBaseTrait("jolokia", 1800),
		JolokiaTrait: traitv1.JolokiaTrait{
			Port: 8778,
		},
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil
}

func (t *jolokiaTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Add the Camel management and Jolokia agent dependencies
		// Also add the Camel JAXB dependency, that's required by Hawtio

		if e.CamelCatalog.Runtime.Provider == v1.RuntimeProviderQuarkus {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "camel-quarkus:management")
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "camel:jaxb")
		}

		// TODO: We may want to make the Jolokia version configurable
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.jolokia:jolokia-jvm:jar:1.7.1")

		return nil
	}

	container := e.GetIntegrationContainer()
	if container == nil {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionJolokiaAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionContainerNotAvailableReason,
			"",
		)
		return nil
	}

	// Configure the Jolokia Java agent, first with the extra options
	options, err := keyValuePairArrayAsStringMap(t.Options)
	if err != nil {
		return err
	}

	t.setDefaultJolokiaOption(options, &t.Host, "host", "*")
	t.setDefaultJolokiaOption(options, &t.DiscoveryEnabled, "discoveryEnabled", false)

	// Configure HTTPS by default for OpenShift
	if e.DetermineProfile() == v1.TraitProfileOpenShift {
		t.setDefaultJolokiaOption(options, &t.Protocol, "protocol", "https")
		t.setDefaultJolokiaOption(options, &t.CaCert, "caCert", "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
		t.setDefaultJolokiaOption(options, &t.ExtendedClientCheck, "extendedClientCheck", true)
		t.setDefaultJolokiaOption(options, &t.UseSslClientAuthentication, "useSslClientAuthentication", true)
		t.setDefaultJolokiaOption(options, &t.ClientPrincipal, "clientPrincipal", []string{
			// Master API proxy for OpenShift 3
			"cn=system:master-proxy",
			// Default Hawtio and Fuse consoles for OpenShift 4
			"cn=hawtio-online.hawtio.svc",
			"cn=fuse-console.fuse.svc",
		})
	}

	// Then add explicitly set trait configuration properties
	t.addToJolokiaOptions(options, "caCert", t.CaCert)
	t.addToJolokiaOptions(options, "clientPrincipal", t.ClientPrincipal)
	t.addToJolokiaOptions(options, "discoveryEnabled", t.DiscoveryEnabled)
	t.addToJolokiaOptions(options, "extendedClientCheck", t.ExtendedClientCheck)
	t.addToJolokiaOptions(options, "host", t.Host)
	t.addToJolokiaOptions(options, "password", t.Password)
	t.addToJolokiaOptions(options, "port", t.Port)
	t.addToJolokiaOptions(options, "protocol", t.Protocol)
	t.addToJolokiaOptions(options, "user", t.User)
	t.addToJolokiaOptions(options, "useSslClientAuthentication", t.UseSslClientAuthentication)

	// Options must be sorted so that the environment variable value is consistent over iterations,
	// otherwise the value changes which results in triggering a new deployment.
	optionValues := make([]string, len(options))
	for i, k := range util.SortedStringMapKeys(options) {
		optionValues[i] = k + "=" + options[k]
	}

	container.Args = append(container.Args, "-javaagent:dependencies/lib/main/org.jolokia.jolokia-jvm-1.7.1.jar="+strings.Join(optionValues, ","))

	containerPort := corev1.ContainerPort{
		Name:          "jolokia",
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionJolokiaAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionJolokiaAvailableReason,
		fmt.Sprintf("%s(%s/%d)", container.Name, containerPort.Name, containerPort.ContainerPort),
	)

	container.Ports = append(container.Ports, containerPort)

	return nil
}

func (t *jolokiaTrait) setDefaultJolokiaOption(options map[string]string, option interface{}, key string, value interface{}) {
	// Do not override existing option
	if _, ok := options[key]; ok {
		return
	}
	switch o := option.(type) {
	case **bool:
		if *o == nil {
			v, _ := value.(bool)
			*o = &v
		}
	case **int:
		if *o == nil {
			v, _ := value.(int)
			*o = &v
		}
	case **string:
		if *o == nil {
			v, _ := value.(string)
			*o = &v
		}
	case *[]string:
		if len(*o) == 0 {
			*o, _ = value.([]string)
		}
	}
}

func (t *jolokiaTrait) addToJolokiaOptions(options map[string]string, key string, value interface{}) {
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
	case string:
		if v != "" {
			options[key] = v
		}
	case []string:
		if len(v) == 1 {
			options[key] = v[0]
		} else {
			for i, vi := range v {
				options[key+"."+strconv.Itoa(i+1)] = vi
			}
		}
	}
}

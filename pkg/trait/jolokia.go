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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// The Jolokia trait activates and configures the Jolokia Java agent.
//
// See https://jolokia.org/reference/html/agents.html
//
// +camel-k:trait=jolokia
type jolokiaTrait struct {
	BaseTrait `property:",squash"`
	// The PEM encoded CA certification file path, used to verify client certificates,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt` for OpenShift).
	CaCert *string `property:"ca-cert" json:"CACert,omitempty"`
	// The principal(s) which must be given in a client certificate to allow access to the Jolokia endpoint,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `clientPrincipal=cn=system:master-proxy`, `cn=hawtio-online.hawtio.svc` and `cn=fuse-console.fuse.svc` for OpenShift).
	ClientPrincipal []string `property:"client-principal" json:"clientPrincipal,omitempty"`
	// Listen for multicast requests (default `false`)
	DiscoveryEnabled *bool `property:"discovery-enabled" json:"discoveryEnabled,omitempty"`
	// Mandate the client certificate contains a client flag in the extended key usage section,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `true` for OpenShift).
	ExtendedClientCheck *bool `property:"extended-client-check" json:"extendedClientCheck,omitempty"`
	// The Host address to which the Jolokia agent should bind to. If `"\*"` or `"0.0.0.0"` is given,
	// the servers binds to every network interface (default `"*"`).
	Host *string `property:"host" json:"host,omitempty"`
	// The password used for authentication, applicable when the `user` option is set.
	Password *string `property:"password" json:"password,omitempty"`
	// The Jolokia endpoint port (default `8778`).
	Port int `property:"port" json:"port,omitempty"`
	// The protocol to use, either `http` or `https` (default `https` for OpenShift)
	Protocol *string `property:"protocol" json:"protocol,omitempty"`
	// The user to be used for authentication
	User *string `property:"user" json:"user,omitempty"`
	// Whether client certificates should be used for authentication (default `true` for OpenShift).
	UseSslClientAuthentication *bool `property:"use-ssl-client-authentication" json:"useSSLClientAuthentication,omitempty"`
	// A list of additional Jolokia options as defined
	// in https://jolokia.org/reference/html/agents.html#agent-jvm-config[JVM agent configuration options]
	Options []string `property:"options" json:"options,omitempty"`
}

func newJolokiaTrait() Trait {
	return &jolokiaTrait{
		BaseTrait: NewBaseTrait("jolokia", 1800),
		Port:      8778,
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, error) {
	return t.Enabled != nil && *t.Enabled && e.IntegrationInPhase(
		v1.IntegrationPhaseInitialization,
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *jolokiaTrait) Apply(e *Environment) (err error) {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Add the Camel management and Jolokia agent dependencies
		// Also add the Camel JAXB dependency, that's required by Hawtio

		switch e.CamelCatalog.Runtime.Provider {
		case v1.RuntimeProviderQuarkus:
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-management")
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "camel-quarkus:jaxb")
		}

		// TODO: We may want to make the Jolokia version configurable
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.jolokia:jolokia-jvm:jar:agent:1.6.2")

		return nil
	}

	container := e.getIntegrationContainer()
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

	container.Args = append(container.Args, "-javaagent:dependencies/org.jolokia.jolokia-jvm-1.6.2-agent.jar="+strings.Join(optionValues, ","))

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
	case *[]string:
		if len(*o) == 0 {
			*o = value.([]string)
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

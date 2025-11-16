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
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
)

const (
	jolokiaTraitID    = "jolokia"
	jolokiaTraitOrder = 1800

	defaultJolokiaPort = 8778
	trueString         = "true"
)

type jolokiaTrait struct {
	BaseTrait
	traitv1.JolokiaTrait `property:",squash"`
}

func newJolokiaTrait() Trait {
	return &jolokiaTrait{
		BaseTrait: NewBaseTrait(jolokiaTraitID, jolokiaTraitOrder),
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	condition := NewIntegrationCondition(
		"Jolokia",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		TraitConfigurationReason,
		"Jolokia trait is deprecated in favour of jvm.agents. It may be removed in future version.",
	)

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), condition, nil
}

func (t *jolokiaTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityJolokia)
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

	t.setDefaultJolokiaOption(options, e.DetermineProfile())
	// Then add explicitly set trait configuration properties
	t.addToJolokiaOptions(options)

	// Options must be sorted so that the environment variable value is consistent over iterations,
	// otherwise the value changes which results in triggering a new deployment.
	optionValues := make([]string, len(options))
	for i, k := range util.SortedStringMapKeys(options) {
		optionValues[i] = k + "=" + options[k]
	}

	jolokiaFilepath := ""
	for _, ar := range e.IntegrationKit.Status.Artifacts {
		if strings.HasPrefix(ar.ID, "org.jolokia.jolokia-agent-jvm") || strings.HasPrefix(ar.ID, "org.jolokia.jolokia-jvm") {
			jolokiaFilepath = ar.Target
			break
		}
	}
	container.Args = append(container.Args, "-javaagent:"+jolokiaFilepath+"="+strings.Join(optionValues, ","))
	container.Args = append(container.Args, "-cp", jolokiaFilepath)

	containerPort := corev1.ContainerPort{
		Name:          "jolokia",
		ContainerPort: t.getPort(),
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

func (t *jolokiaTrait) getPort() int32 {
	if t.Port == 0 {
		return defaultJolokiaPort
	}

	return t.Port
}

func (t *jolokiaTrait) setDefaultJolokiaOption(options map[string]string, profile v1.TraitProfile) {
	// Do not override existing option
	if options["host"] == "" {
		options["host"] = "*"
	}
	if options["discoveryEnabled"] == "" {
		options["discoveryEnabled"] = "false"
	}
	//nolint:nestif
	if profile == v1.TraitProfileOpenShift {
		if options["protocol"] == "" {
			options["protocol"] = "https"
		}
		if options["caCert"] == "" {
			options["caCert"] = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
		}
		if options["extendedClientCheck"] == "" {
			options["extendedClientCheck"] = trueString
		}
		if options["useSslClientAuthentication"] == "" {
			options["useSslClientAuthentication"] = trueString
		}
		if options["clientPrincipal.1"] == "" {
			options["clientPrincipal.1"] = "cn=system:master-proxy"
		}
		if options["clientPrincipal.2"] == "" {
			options["clientPrincipal.2"] = "cn=hawtio-online.hawtio.svc"
		}
		if options["clientPrincipal.3"] == "" {
			options["clientPrincipal.3"] = "cn=fuse-console.fuse.svc"
		}
	}
}

func (t *jolokiaTrait) addToJolokiaOptions(options map[string]string) {
	if t.CaCert != nil {
		options["caCert"] = *t.CaCert
	}
	if t.ClientPrincipal != nil {
		for i, v := range t.ClientPrincipal {
			options[fmt.Sprintf("clientPrincipal.%v", i)] = v
		}
	}
	if t.ExtendedClientCheck != nil {
		options["extendedClientCheck"] = strconv.FormatBool(*t.ExtendedClientCheck)
	}
	if t.Host != nil {
		options["host"] = *t.Host
	}
	if t.Password != nil {
		options["password"] = *t.Password
	}
	options["port"] = strconv.Itoa(int(t.getPort()))
	if t.Protocol != nil {
		options["protocol"] = *t.Protocol
	}
	if t.User != nil {
		options["user"] = *t.User
	}
	if t.UseSslClientAuthentication != nil {
		options["useSslClientAuthentication"] = strconv.FormatBool(*t.UseSslClientAuthentication)
	}
}

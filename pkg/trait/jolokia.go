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
	"errors"
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"

	"github.com/sirupsen/logrus"
)

type jolokiaTrait struct {
	BaseTrait `property:",squash"`

	OpenShiftSSLAuth *bool   `property:"openshift-ssl-auth"`
	Options          *string `property:"options"`
	Port             int     `property:"port"`
	RandomPassword   *bool   `property:"random-password"`
}

// The Jolokia trait must be executed prior to the deployment trait
// as it mutates environment variables
func newJolokiaTrait() *jolokiaTrait {
	return &jolokiaTrait{
		BaseTrait: BaseTrait{
			id: ID("jolokia"),
		},
		Port: 8778,
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, error) {
	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

// Configure the Jolokia Java agent
func (t *jolokiaTrait) Apply(e *Environment) (err error) {
	if t.Enabled == nil || !*t.Enabled {
		// Deactivate the Jolokia Java agent
		// Note: the AB_JOLOKIA_OFF environment variable acts as an option flag
		envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_OFF", "true")
		return nil
	}

	// OpenShift proxy SSL client authentication
	if e.DetermineProfile() == v1alpha1.TraitProfileOpenShift {
		if t.OpenShiftSSLAuth != nil && !*t.OpenShiftSSLAuth {
			envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_AUTH_OPENSHIFT", "false")
		} else {
			envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_AUTH_OPENSHIFT", "true")
		}
	} else {
		if t.OpenShiftSSLAuth != nil {
			logrus.Warn("Jolokia trait property [openshiftSSLAuth] is only applicable for the OpenShift profile!")
		}
		envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_AUTH_OPENSHIFT", "false")
	}
	// Jolokia options
	if t.Options != nil {
		envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_OPTS", *t.Options)
	}
	// Agent port
	envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_PORT", strconv.Itoa(t.Port))
	// Random password
	if t.RandomPassword != nil {
		envvar.SetVal(&e.EnvVars, "AB_JOLOKIA_PASSWORD_RANDOM", strconv.FormatBool(*t.RandomPassword))
	}

	// Register a post processor to add a container port to the integration deployment
	e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
		var container *corev1.Container
		environment.Resources.VisitContainer(func(c *corev1.Container) {
			if c.Name == environment.Integration.Name {
				container = c
			}
		})
		if container != nil {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          "jolokia",
				ContainerPort: int32(t.Port),
				Protocol:      corev1.ProtocolTCP,
			})
		} else {
			return errors.New("Cannot add Jolokia container port: no integration container")
		}
		return nil
	})

	return nil
}

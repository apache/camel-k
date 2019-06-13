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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type serviceTrait struct {
	BaseTrait `property:",squash"`

	Auto *bool `property:"auto"`
	Port int   `property:"port"`
}

const httpPortName = "http"

func newServiceTrait() *serviceTrait {
	return &serviceTrait{
		BaseTrait: newBaseTrait("service"),
		Port:      8080,
	}
}

func (t *serviceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		sources, err := kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources)
		if err != nil {
			return false, err
		}
		meta := metadata.ExtractAll(e.CamelCatalog, sources)
		if !meta.RequiresHTTPService {
			return false, nil
		}
	}

	return true, nil
}

func (t *serviceTrait) Apply(e *Environment) (err error) {
	// Either update the existing service added by previously executed traits
	// (e.g. the prometheus trait) or add a new service resource
	svc := e.Resources.GetService(func(svc *corev1.Service) bool {
		return svc.Name == e.Integration.Name
	})
	if svc == nil {
		svc = getServiceFor(e)
		e.Resources.Add(svc)
	}
	port := corev1.ServicePort{
		Name:       httpPortName,
		Port:       80,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(httpPortName),
	}
	svc.Spec.Ports = append(svc.Spec.Ports, port)

	// Mark the service as a user service
	svc.Labels["camel.apache.org/service.type"] = "user"

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
				Name:          httpPortName,
				ContainerPort: int32(t.Port),
				Protocol:      corev1.ProtocolTCP,
			})
		} else {
			return errors.New("Cannot add HTTP container port: no integration container")
		}
		return nil
	})

	return nil
}

func getServiceFor(e *Environment) *corev1.Service {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
			Selector: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
	}

	return &svc
}

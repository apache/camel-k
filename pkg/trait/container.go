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
	"strings"

	"github.com/apache/camel-k/pkg/util/envvar"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultContainerName = "integration"
	containerTraitID     = "container"
)

type containerTrait struct {
	BaseTrait `property:",squash"`

	Auto            *bool  `property:"auto"`
	RequestCPU      string `property:"request-cpu"`
	RequestMemory   string `property:"request-memory"`
	LimitCPU        string `property:"limit-cpu"`
	LimitMemory     string `property:"limit-memory"`
	Expose          *bool  `property:"expose"`
	Port            int    `property:"port"`
	PortName        string `property:"port-name"`
	ServicePort     int    `property:"service-port"`
	ServicePortName string `property:"service-port-name"`
	Name            string `property:"name"`
}

func newContainerTrait() *containerTrait {
	return &containerTrait{
		BaseTrait:       newBaseTrait(containerTraitID),
		Port:            8080,
		PortName:        httpPortName,
		ServicePort:     80,
		ServicePortName: httpPortName,
		Name:            defaultContainerName,
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.Expose == nil {
			e := e.Resources.GetServiceForIntegration(e.Integration) != nil
			t.Expose = &e
		}
	}

	return true, nil
}

func (t *containerTrait) Apply(e *Environment) error {
	container := corev1.Container{
		Name:  t.Name,
		Image: e.Integration.Status.Image,
		Env:   make([]corev1.EnvVar, 0),
	}

	// combine Environment of integration with platform, kit, integration
	for key, value := range e.CollectConfigurationPairs("env") {
		envvar.SetVal(&container.Env, key, value)
	}

	envvar.SetVal(&container.Env, "CAMEL_K_DIGEST", e.Integration.Status.Digest)
	envvar.SetVal(&container.Env, "CAMEL_K_ROUTES", strings.Join(e.ComputeSourcesURI(), ","))
	envvar.SetVal(&container.Env, "CAMEL_K_CONF", "/etc/camel/conf/application.properties")
	envvar.SetVal(&container.Env, "CAMEL_K_CONF_D", "/etc/camel/conf.d")

	t.configureResources(e, &container)

	e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}

		e.ConfigureVolumesAndMounts(
			&deployment.Spec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
	})

	e.Resources.VisitKnativeService(func(service *serving.Service) {
		for _, env := range e.EnvVars {
			switch {
			case env.ValueFrom == nil:
				envvar.SetVar(&container.Env, env)
			case env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.FieldPath == "metadata.namespace":
				envvar.SetVar(&container.Env, corev1.EnvVar{Name: env.Name, Value: e.Integration.Namespace})
			case env.ValueFrom.FieldRef != nil:
				t.L.Infof("Environment variable %s uses fieldRef and cannot be set on a Knative service", env.Name)
			case env.ValueFrom.ResourceFieldRef != nil:
				t.L.Infof("Environment variable %s uses resourceFieldRef and cannot be set on a Knative service", env.Name)
			default:
				envvar.SetVar(&container.Env, env)
			}
		}

		e.ConfigureVolumesAndMounts(
			&service.Spec.ConfigurationSpec.GetTemplate().Spec.Volumes,
			&container.VolumeMounts,
		)

		service.Spec.ConfigurationSpec.GetTemplate().Spec.Containers = append(service.Spec.ConfigurationSpec.GetTemplate().Spec.Containers , container)
	})

	if t.Expose != nil && *t.Expose {
		t.configureService(e)
	}

	return nil
}

func (t *containerTrait) configureService(e *Environment) {
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		return
	}

	container := e.Resources.GetContainerByName(t.Name)
	if container == nil {
		return
	}

	containerPort := corev1.ContainerPort{
		Name:          t.PortName,
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	servicePort := corev1.ServicePort{
		Name:       t.ServicePortName,
		Port:       int32(t.ServicePort),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(t.PortName),
	}

	e.Integration.Status.SetCondition(
		v1alpha1.IntegrationConditionServiceAvailable,
		corev1.ConditionTrue,
		v1alpha1.IntegrationConditionServiceAvailableReason,

		// service -> container
		fmt.Sprintf("%s(%s/%d) -> %s(%s/%d)",
			service.Name, servicePort.Name, servicePort.Port,
			container.Name, containerPort.Name, containerPort.ContainerPort),
	)

	container.Ports = append(container.Ports, containerPort)
	service.Spec.Ports = append(service.Spec.Ports, servicePort)

	// Mark the service as a user service
	service.Labels["camel.apache.org/service.type"] = v1alpha1.ServiceTypeUser
}

func (t *containerTrait) configureResources(_ *Environment, container *corev1.Container) {
	//
	// Requests
	//
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	if t.RequestCPU != "" {
		v, err := resource.ParseQuantity(t.RequestCPU)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "request-cpu", t.RequestCPU)
		} else {
			container.Resources.Requests[corev1.ResourceCPU] = v
		}
	}
	if t.RequestMemory != "" {
		v, err := resource.ParseQuantity(t.RequestMemory)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "request-memory", t.RequestMemory)
		} else {
			container.Resources.Requests[corev1.ResourceMemory] = v
		}
	}

	//
	// Limits
	//
	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}

	if t.LimitCPU != "" {
		v, err := resource.ParseQuantity(t.LimitCPU)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "limit-cpu", t.LimitCPU)
		} else {
			container.Resources.Limits[corev1.ResourceCPU] = v
		}
	}
	if t.LimitMemory != "" {
		v, err := resource.ParseQuantity(t.LimitMemory)
		if err != nil {
			t.L.Error(err, "unable to parse quantity", "limit-memory", t.LimitMemory)
		} else {
			container.Resources.Limits[corev1.ResourceMemory] = v
		}
	}
}

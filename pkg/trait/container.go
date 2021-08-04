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
	"path"
	"sort"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
)

const (
	defaultContainerName     = "integration"
	defaultContainerPort     = 8080
	defaultContainerPortName = "http"
	defaultServicePort       = 80
	defaultProbePath         = "/q/health"
	containerTraitID         = "container"
)

// The Container trait can be used to configure properties of the container where the integration will run.
//
// It also provides configuration for Services associated to the container.
//
// +camel-k:trait=container
type containerTrait struct {
	BaseTrait `property:",squash"`

	Auto *bool `property:"auto" json:"auto,omitempty"`

	// The minimum amount of CPU required.
	RequestCPU string `property:"request-cpu" json:"requestCPU,omitempty"`
	// The minimum amount of memory required.
	RequestMemory string `property:"request-memory" json:"requestMemory,omitempty"`
	// The maximum amount of CPU required.
	LimitCPU string `property:"limit-cpu" json:"limitCPU,omitempty"`
	// The maximum amount of memory required.
	LimitMemory string `property:"limit-memory" json:"limitMemory,omitempty"`

	// Can be used to enable/disable exposure via kubernetes Service.
	Expose *bool `property:"expose" json:"expose,omitempty"`
	// To configure a different port exposed by the container (default `8080`).
	Port int `property:"port" json:"port,omitempty"`
	// To configure a different port name for the port exposed by the container (default `http`).
	PortName string `property:"port-name" json:"portName,omitempty"`
	// To configure under which service port the container port is to be exposed (default `80`).
	ServicePort int `property:"service-port" json:"servicePort,omitempty"`
	// To configure under which service port name the container port is to be exposed (default `http`).
	ServicePortName string `property:"service-port-name" json:"servicePortName,omitempty"`

	// The main container name. It's named `integration` by default.
	Name string `property:"name" json:"name,omitempty"`
	// The main container image
	Image string `property:"image" json:"image,omitempty"`

	// ProbesEnabled enable/disable probes on the container (default `false`)
	ProbesEnabled *bool `property:"probes-enabled" json:"probesEnabled,omitempty"`
	// Number of seconds after the container has started before liveness probes are initiated.
	LivenessInitialDelay int32 `property:"liveness-initial-delay" json:"livenessInitialDelay,omitempty"`
	// Number of seconds after which the probe times out. Applies to the liveness probe.
	LivenessTimeout int32 `property:"liveness-timeout" json:"livenessTimeout,omitempty"`
	// How often to perform the probe. Applies to the liveness probe.
	LivenessPeriod int32 `property:"liveness-period" json:"livenessPeriod,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the liveness probe.
	LivenessSuccessThreshold int32 `property:"liveness-success-threshold" json:"livenessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the liveness probe.
	LivenessFailureThreshold int32 `property:"liveness-failure-threshold" json:"livenessFailureThreshold,omitempty"`
	// Number of seconds after the container has started before readiness probes are initiated.
	ReadinessInitialDelay int32 `property:"readiness-initial-delay" json:"readinessInitialDelay,omitempty"`
	// Number of seconds after which the probe times out. Applies to the readiness probe.
	ReadinessTimeout int32 `property:"readiness-timeout" json:"readinessTimeout,omitempty"`
	// How often to perform the probe. Applies to the readiness probe.
	ReadinessPeriod int32 `property:"readiness-period" json:"readinessPeriod,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Applies to the readiness probe.
	ReadinessSuccessThreshold int32 `property:"readiness-success-threshold" json:"readinessSuccessThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Applies to the readiness probe.
	ReadinessFailureThreshold int32 `property:"readiness-failure-threshold" json:"readinessFailureThreshold,omitempty"`
}

func newContainerTrait() Trait {
	return &containerTrait{
		BaseTrait:       NewBaseTrait(containerTraitID, 1600),
		Port:            defaultContainerPort,
		ServicePort:     defaultServicePort,
		ServicePortName: defaultContainerPortName,
		Name:            defaultContainerName,
		ProbesEnabled:   util.BoolP(false),
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if util.IsNilOrTrue(t.Auto) {
		if t.Expose == nil {
			e := e.Resources.GetServiceForIntegration(e.Integration) != nil
			t.Expose = &e
		}
	}

	return true, nil
}

func (t *containerTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		return t.configureDependencies(e)
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return t.configureContainer(e)
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *containerTrait) IsPlatformTrait() bool {
	return true
}

func (t *containerTrait) configureDependencies(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		if t.Image != "" {
			if e.Integration.Spec.IntegrationKit != nil {
				return fmt.Errorf(
					"unsupported configuration: a container image has been set in conjunction with an IntegrationKit %v",
					e.Integration.Spec.IntegrationKit)
			}
			if e.Integration.Spec.Kit != "" {
				return fmt.Errorf(
					"unsupported configuration: a container image has been set in conjunction with an IntegrationKit %s",
					e.Integration.Spec.Kit)
			}

			kitName := fmt.Sprintf("kit-%s", e.Integration.Name)
			kit := v1.NewIntegrationKit(e.Integration.Namespace, kitName)
			kit.Spec.Image = t.Image

			// Add some information for post-processing, this may need to be refactored
			// to a proper data structure
			kit.Labels = map[string]string{
				"camel.apache.org/kit.type":           v1.IntegrationKitTypeExternal,
				kubernetes.CamelCreatorLabelKind:      v1.IntegrationKind,
				kubernetes.CamelCreatorLabelName:      e.Integration.Name,
				kubernetes.CamelCreatorLabelNamespace: e.Integration.Namespace,
				kubernetes.CamelCreatorLabelVersion:   e.Integration.ResourceVersion,
			}

			t.L.Infof("image %s", kit.Spec.Image)
			e.Resources.Add(&kit)
			e.Integration.SetIntegrationKit(&kit)
		}
		if util.IsTrue(t.ProbesEnabled) {
			if capability, ok := e.CamelCatalog.Runtime.Capabilities[v1.CapabilityHealth]; ok {
				for _, dependency := range capability.Dependencies {
					util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dependency.GetDependencyID())
				}

				// sort the dependencies to get always the same list if they don't change
				sort.Strings(e.Integration.Status.Dependencies)
			}
		}
	}

	return nil
}

// nolint:gocyclo
func (t *containerTrait) configureContainer(e *Environment) error {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}

	container := corev1.Container{
		Name:  t.Name,
		Image: e.Integration.Status.Image,
		Env:   make([]corev1.EnvVar, 0),
	}

	// combine Environment of integration with platform, kit, integration
	for _, env := range e.collectConfigurationPairs("env") {
		envvar.SetVal(&container.Env, env.Name, env.Value)
	}

	envvar.SetVal(&container.Env, "CAMEL_K_DIGEST", e.Integration.Status.Digest)
	envvar.SetVal(&container.Env, "CAMEL_K_CONF", path.Join(basePath, "application.properties"))
	envvar.SetVal(&container.Env, "CAMEL_K_CONF_D", confDPath)

	e.addSourcesProperties()

	t.configureResources(e, &container)

	if t.Expose != nil && *t.Expose {
		t.configureService(e, &container)
	}
	t.configureCapabilities(e)

	portName := t.PortName
	if portName == "" {
		portName = defaultContainerPortName
	}
	// Deployment
	if err := e.Resources.VisitDeploymentE(func(deployment *appsv1.Deployment) error {
		if util.IsTrue(t.ProbesEnabled) && portName == defaultContainerPortName {
			t.configureProbes(&container, t.Port, defaultProbePath)
		}

		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		if props, err := e.computeApplicationProperties(); err != nil {
			return err
		} else if props != nil {
			e.Resources.Add(props)
		}

		e.configureVolumesAndMounts(
			&deployment.Spec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	// Knative Service
	if err := e.Resources.VisitKnativeServiceE(func(service *serving.Service) error {
		if util.IsTrue(t.ProbesEnabled) && portName == defaultContainerPortName {
			// don't set the port on Knative service as it is not allowed.
			t.configureProbes(&container, 0, defaultProbePath)
		}

		for _, env := range e.EnvVars {
			switch {
			case env.ValueFrom == nil:
				envvar.SetVar(&container.Env, env)
			case env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.FieldPath == "metadata.namespace":
				envvar.SetVar(&container.Env, corev1.EnvVar{Name: env.Name, Value: e.Integration.Namespace})
			case env.ValueFrom.FieldRef != nil:
				t.L.Infof("Skipping environment variable %s (fieldRef)", env.Name)
			case env.ValueFrom.ResourceFieldRef != nil:
				t.L.Infof("Skipping environment variable %s (resourceFieldRef)", env.Name)
			default:
				envvar.SetVar(&container.Env, env)
			}
		}
		if props, err := e.computeApplicationProperties(); err != nil {
			return err
		} else if props != nil {
			e.Resources.Add(props)
		}

		e.configureVolumesAndMounts(
			&service.Spec.ConfigurationSpec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		service.Spec.ConfigurationSpec.Template.Spec.Containers = append(service.Spec.ConfigurationSpec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	// CronJob
	if err := e.Resources.VisitCronJobE(func(cron *v1beta1.CronJob) error {
		if util.IsTrue(t.ProbesEnabled) && portName == defaultContainerPortName {
			t.configureProbes(&container, t.Port, defaultProbePath)
		}

		for _, envVar := range e.EnvVars {
			envvar.SetVar(&container.Env, envVar)
		}
		if props, err := e.computeApplicationProperties(); err != nil {
			return err
		} else if props != nil {
			e.Resources.Add(props)
		}

		e.configureVolumesAndMounts(
			&cron.Spec.JobTemplate.Spec.Template.Spec.Volumes,
			&container.VolumeMounts,
		)

		cron.Spec.JobTemplate.Spec.Template.Spec.Containers = append(cron.Spec.JobTemplate.Spec.Template.Spec.Containers, container)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (t *containerTrait) configureService(e *Environment, container *corev1.Container) {
	service := e.Resources.GetServiceForIntegration(e.Integration)
	if service == nil {
		return
	}

	name := t.PortName
	if name == "" {
		name = defaultContainerPortName
	}

	containerPort := corev1.ContainerPort{
		Name:          name,
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	servicePort := corev1.ServicePort{
		Name:       t.ServicePortName,
		Port:       int32(t.ServicePort),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(name),
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionServiceAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionServiceAvailableReason,

		// service -> container
		fmt.Sprintf("%s(%s/%d) -> %s(%s/%d)",
			service.Name, servicePort.Name, servicePort.Port,
			container.Name, containerPort.Name, containerPort.ContainerPort),
	)

	container.Ports = append(container.Ports, containerPort)
	service.Spec.Ports = append(service.Spec.Ports, servicePort)

	// Mark the service as a user service
	service.Labels["camel.apache.org/service.type"] = v1.ServiceTypeUser
}

func (t *containerTrait) configureResources(_ *Environment, container *corev1.Container) {
	// Requests
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

	// Limits
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

func (t *containerTrait) configureCapabilities(e *Environment) {
	if util.StringSliceExists(e.Integration.Status.Capabilities, v1.CapabilityRest) {
		e.ApplicationProperties["camel.context.rest-configuration.component"] = "platform-http"
	}
}

func (t *containerTrait) configureProbes(container *corev1.Container, port int, path string) {
	container.LivenessProbe = t.newLivenessProbe(port, path)
	container.ReadinessProbe = t.newReadinessProbe(port, path)
}

func (t *containerTrait) newLivenessProbe(port int, path string) *corev1.Probe {
	action := corev1.HTTPGetAction{}
	action.Path = path

	if port > 0 {
		action.Port = intstr.FromInt(port)
	}

	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &action,
		},
	}

	p.InitialDelaySeconds = t.LivenessInitialDelay
	p.TimeoutSeconds = t.LivenessTimeout
	p.PeriodSeconds = t.LivenessPeriod
	p.SuccessThreshold = t.LivenessSuccessThreshold
	p.FailureThreshold = t.LivenessFailureThreshold

	return &p
}

func (t *containerTrait) newReadinessProbe(port int, path string) *corev1.Probe {
	p := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(port),
				Path: path,
			},
		},
	}

	p.InitialDelaySeconds = t.ReadinessInitialDelay
	p.TimeoutSeconds = t.ReadinessTimeout
	p.PeriodSeconds = t.ReadinessPeriod
	p.SuccessThreshold = t.ReadinessSuccessThreshold
	p.FailureThreshold = t.ReadinessFailureThreshold

	return &p
}
